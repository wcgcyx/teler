package rpc

/*
 * Licensed under LGPL-3.0.
 *
 * You can get a copy of the LGPL-3.0 License at
 *
 * https://www.gnu.org/licenses/lgpl-3.0.en.html
 *
 * @wcgcyx - https://github.com/wcgcyx
 */

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/wcgcyx/teler/backend"
	"github.com/wcgcyx/teler/processor"
	"github.com/wcgcyx/teler/worldstate"
)

// Note:
// This is adapted from:
// 		go-ethereum@v1.15.2/eth/tracers/api.go

const (
	// defaultTraceTimeout is the amount of time a single transaction can execute
	// by default before being forcefully aborted.
	defaultTraceTimeout = 5 * time.Second

	// defaultTraceReexec is the number of blocks the tracer is willing to go back
	// and reexecute to produce missing historical state necessary to run a specific
	// trace.
	defaultTraceReexec = uint64(128)

	// // defaultTracechainMemLimit is the size of the triedb, at which traceChain
	// // switches over and tries to use a disk-backed database instead of building
	// // on top of memory.
	// // For non-archive nodes, this limit _will_ be overblown, as disk-backed tries
	// // will only be found every ~15K blocks or so.
	// defaultTracechainMemLimit = common.StorageSize(500 * 1024 * 1024)

	// // maximumPendingTraceStates is the maximum number of states allowed waiting
	// // for tracing. The creation of trace state will be paused if the unused
	// // trace states exceed this limit.
	// maximumPendingTraceStates = 128
)

// TODO: Release resource after tracing is done, as in the form of tracers.StateReleaseFunc

// txTraceResult is the result of a single transaction trace.
type txTraceResult struct {
	TxHash common.Hash `json:"txHash"`           // transaction hash
	Result interface{} `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string      `json:"error,omitempty"`  // Trace failure produced by the tracer
}

// txTraceTask represents a single transaction trace task when an entire block
// is being traced.
type txTraceTask struct {
	state worldstate.MutableWorldState // Intermediate state prepped for tracing
	index int                          // Transaction offset in the block
}

// TraceCallConfig is the config for traceCall API. It holds one more
// field to override the state for tracing.
type TraceCallConfig struct {
	tracers.TraceConfig
	StateOverrides *StateOverride
	BlockOverrides *BlockOverrides
	TxIndex        *hexutil.Uint
}

// traceBlock configures a new tracer according to the provided configuration, and
// executes all the transactions contained within. The return value will be one item
// per transaction, dependent on the requested tracer.
func traceBlock(ctx context.Context, be backend.Backend, block *types.Block, config *tracers.TraceConfig) ([]*txTraceResult, error) {
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	parent, err := be.Blockchain().GetHeaderByHash(ctx, block.ParentHash())
	if err != nil {
		return nil, err
	}
	// Note: reexec is ignored as state not available has been digested.
	// reexec := defaultTraceReexec
	// if config != nil && config.Reexec != nil {
	// 	reexec = *config.Reexec
	// }
	state, err := be.StateArchive().GetMutable(parent.Number.Uint64(), parent.Root)
	if err != nil {
		return nil, err
	}
	// JS tracers have high overhead. In this case run a parallel
	// process that generates states in one thread and traces txes
	// in separate worker threads.
	if config != nil && config.Tracer != nil && *config.Tracer != "" {
		if isJS := tracers.DefaultDirectory.IsJS(*config.Tracer); isJS {
			traceBlockParallel(ctx, be, block, state, config)
			return nil, nil
		}
	}
	// Native tracers have low overhead
	var (
		txs       = block.Transactions()
		blockHash = block.Hash()
		blockCtx  = core.NewEVMBlockContext(block.Header(), processor.NewChainContext(ctx, be.Blockchain(), be.Processor().Engine, be.ChainConfig()), nil)
		signer    = types.MakeSigner(be.ChainConfig(), block.Number(), block.Time())
		results   = make([]*txTraceResult, len(txs))
	)
	if beaconRoot := block.BeaconRoot(); beaconRoot != nil {
		vmenv := vm.NewEVM(blockCtx, state, be.ChainConfig(), vm.Config{})
		core.ProcessBeaconBlockRoot(*beaconRoot, vmenv)
	}
	for i, tx := range txs {
		// Generate the next state snapshot fast without tracing
		msg, _ := core.TransactionToMessage(tx, signer, block.BaseFee())
		txctx := &tracers.Context{
			BlockHash:   blockHash,
			BlockNumber: block.Number(),
			TxIndex:     i,
			TxHash:      tx.Hash(),
		}
		res, err := traceTx(ctx, be, tx, msg, txctx, blockCtx, state, config)
		if err != nil {
			return nil, err
		}
		results[i] = &txTraceResult{TxHash: tx.Hash(), Result: res}
	}
	return results, nil
}

// traceTx configures a new tracer according to the provided configuration, and
// executes the given message in the provided environment. The return value will
// be tracer dependent.
func traceTx(ctx context.Context, be backend.Backend, tx *types.Transaction, message *core.Message, txctx *tracers.Context, vmctx vm.BlockContext, state worldstate.MutableWorldState, config *tracers.TraceConfig) (interface{}, error) {
	var (
		tracer  *tracers.Tracer
		err     error
		timeout = defaultTraceTimeout
		usedGas uint64
	)
	if config == nil {
		config = &tracers.TraceConfig{}
	}
	// Default tracer is the struct logger
	if config.Tracer == nil {
		logger := logger.NewStructLogger(config.Config)
		tracer = &tracers.Tracer{
			Hooks:     logger.Hooks(),
			GetResult: logger.GetResult,
			Stop:      logger.Stop,
		}
	} else {
		tracer, err = tracers.DefaultDirectory.New(*config.Tracer, txctx, config.TracerConfig, be.ChainConfig())
		if err != nil {
			return nil, err
		}
	}
	// The actual TxContext will be created as part of ApplyTransactionWithEVM.
	vmenv := vm.NewEVM(vmctx, state, be.ChainConfig(), vm.Config{Tracer: tracer.Hooks, NoBaseFee: true})
	state.SetLogger(tracer.Hooks)

	// Define a meaningful timeout of a single transaction trace
	if config.Timeout != nil {
		if timeout, err = time.ParseDuration(*config.Timeout); err != nil {
			return nil, err
		}
	}
	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	go func() {
		<-deadlineCtx.Done()
		if errors.Is(deadlineCtx.Err(), context.DeadlineExceeded) {
			tracer.Stop(errors.New("execution timeout"))
			// Stop evm execution. Note cancellation is not necessarily immediate.
			vmenv.Cancel()
		}
	}()
	defer cancel()

	// Call Prepare to clear out the statedb access list
	state.SetTxContext(txctx.TxHash, txctx.TxIndex)
	_, err = processor.ApplyTransaction(message, new(core.GasPool).AddGas(message.GasLimit), state, vmctx.BlockNumber, txctx.BlockHash, tx, &usedGas, vmenv)
	if err != nil {
		return nil, fmt.Errorf("tracing failed: %w", err)
	}
	return tracer.GetResult()
}

// traceBlockParallel is for tracers that have a high overhead (read JS tracers). One thread
// runs along and executes txes without tracing enabled to generate their prestate.
// Worker threads take the tasks and the prestate and trace them.
func traceBlockParallel(ctx context.Context, be backend.Backend, block *types.Block, state worldstate.MutableWorldState, config *tracers.TraceConfig) ([]*txTraceResult, error) {
	// Execute all the transaction contained within the block concurrently
	var (
		txs       = block.Transactions()
		blockHash = block.Hash()
		signer    = types.MakeSigner(be.ChainConfig(), block.Number(), block.Time())
		results   = make([]*txTraceResult, len(txs))
		pend      sync.WaitGroup
	)
	threads := runtime.NumCPU()
	if threads > len(txs) {
		threads = len(txs)
	}
	jobs := make(chan *txTraceTask, threads)
	for th := 0; th < threads; th++ {
		pend.Add(1)
		go func() {
			defer pend.Done()
			// Fetch and execute the next transaction trace tasks
			for task := range jobs {
				msg, _ := core.TransactionToMessage(txs[task.index], signer, block.BaseFee())
				txctx := &tracers.Context{
					BlockHash:   blockHash,
					BlockNumber: block.Number(),
					TxIndex:     task.index,
					TxHash:      txs[task.index].Hash(),
				}
				// Reconstruct the block context for each transaction
				// as the GetHash function of BlockContext is not safe for
				// concurrent use.
				// See: https://github.com/ethereum/go-ethereum/issues/29114
				blockCtx := core.NewEVMBlockContext(block.Header(), processor.NewChainContext(ctx, be.Blockchain(), be.Processor().Engine, be.ChainConfig()), nil)
				res, err := traceTx(ctx, be, txs[task.index], msg, txctx, blockCtx, state, config)
				if err != nil {
					results[task.index] = &txTraceResult{TxHash: txs[task.index].Hash(), Error: err.Error()}
					continue
				}
				results[task.index] = &txTraceResult{TxHash: txs[task.index].Hash(), Result: res}
			}
		}()
	}

	// Feed the transactions into the tracers and return
	var failed error
	blockCtx := core.NewEVMBlockContext(block.Header(), processor.NewChainContext(ctx, be.Blockchain(), be.Processor().Engine, be.ChainConfig()), nil)
txloop:
	for i, tx := range txs {
		// Send the trace task over for execution
		task := &txTraceTask{state: state.Copy(), index: i}
		select {
		case <-ctx.Done():
			failed = ctx.Err()
			break txloop
		case jobs <- task:
		}

		// Generate the next state snapshot fast without tracing
		msg, _ := core.TransactionToMessage(tx, signer, block.BaseFee())
		state.SetTxContext(tx.Hash(), i)
		vmenv := vm.NewEVM(blockCtx, state, be.ChainConfig(), vm.Config{})
		if _, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(msg.GasLimit)); err != nil {
			failed = err
			break txloop
		}
		// Finalize the state so any modifications are written to the trie
		// Only delete empty objects if EIP158/161 (a.k.a Spurious Dragon) is in effect
		state.Finalise(vmenv.ChainConfig().IsEIP158(block.Number()))
	}

	close(jobs)
	pend.Wait()

	// If execution failed in between, abort
	if failed != nil {
		return nil, failed
	}
	return results, nil
}

// stateAtTransaction returns the execution environment of a certain transaction.
func stateAtTransaction(ctx context.Context, be backend.Backend, block *types.Block, txIndex int, _ uint64) (*types.Transaction, vm.BlockContext, worldstate.MutableWorldState, tracers.StateReleaseFunc, error) {
	// Short circuit if it's genesis block.
	if block.NumberU64() == 0 {
		return nil, vm.BlockContext{}, nil, nil, errors.New("no transaction in genesis")
	}
	// Create the parent state database
	parent, err := be.Blockchain().GetHeaderByHash(ctx, block.ParentHash())
	if err != nil {
		return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("parent %#x not found", block.ParentHash())
	}
	// Lookup the statedb of parent block from the live database,
	// otherwise regenerate it on the flight.
	state, err := be.StateArchive().GetMutable(parent.Number.Uint64(), parent.Root)
	if err != nil {
		return nil, vm.BlockContext{}, nil, nil, err
	}
	// Insert parent beacon block root in the state as per EIP-4788.
	if beaconRoot := block.BeaconRoot(); beaconRoot != nil {
		context := core.NewEVMBlockContext(block.Header(), processor.NewChainContext(ctx, be.Blockchain(), be.Processor().Engine, be.ChainConfig()), nil)
		vmenv := vm.NewEVM(context, state, be.ChainConfig(), vm.Config{})
		core.ProcessBeaconBlockRoot(*beaconRoot, vmenv)
	}
	if txIndex == 0 && len(block.Transactions()) == 0 {
		return nil, vm.BlockContext{}, state, func() {}, nil
	}
	// Recompute transactions up to the target index.
	signer := types.MakeSigner(be.ChainConfig(), block.Number(), block.Time())
	for idx, tx := range block.Transactions() {
		// Assemble the transaction call message and return if the requested offset
		msg, _ := core.TransactionToMessage(tx, signer, block.BaseFee())
		context := core.NewEVMBlockContext(block.Header(), processor.NewChainContext(ctx, be.Blockchain(), be.Processor().Engine, be.ChainConfig()), nil)
		if idx == txIndex {
			return tx, context, state, func() {}, nil
		}
		// Not yet the searched for transaction, execute on top of the current state
		vmenv := vm.NewEVM(context, state, be.ChainConfig(), vm.Config{})
		state.SetTxContext(tx.Hash(), idx)
		if _, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(tx.Gas())); err != nil {
			return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("transaction %#x failed: %v", tx.Hash(), err)
		}
		// Ensure any modifications are committed to the state
		// Only delete empty objects if EIP158/161 (a.k.a Spurious Dragon) is in effect
		state.Finalise(vmenv.ChainConfig().IsEIP158(block.Number()))
	}
	return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("transaction index %d out of range for block %#x", txIndex, block.Hash())
}
