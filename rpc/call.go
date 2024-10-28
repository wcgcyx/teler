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
	"math"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/wcgcyx/teler/backend"
	"github.com/wcgcyx/teler/processor"
	"github.com/wcgcyx/teler/worldstate"
)

// Note:
// This is adapted from:
// 		go-ethereum@v1.14.8/internal/ethapi/api.go

func DoCall(ctx context.Context, be backend.Backend, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *StateOverride, blockOverrides *BlockOverrides, timeout time.Duration, globalGasCap uint64) (*core.ExecutionResult, error) {
	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	layer, header, err := stateAndHeaderByNumberOrHash(ctx, be, blockNrOrHash)
	if layer == nil || err != nil {
		return nil, err
	}

	return doCall(ctx, be, args, layer.GetMutable(), header, overrides, blockOverrides, timeout, globalGasCap)
}

func doCall(ctx context.Context, be backend.Backend, args TransactionArgs, state worldstate.MutableWorldState, header *types.Header, overrides *StateOverride, blockOverrides *BlockOverrides, timeout time.Duration, globalGasCap uint64) (*core.ExecutionResult, error) {
	if err := overrides.Apply(state); err != nil {
		return nil, err
	}
	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	// Get a new instance of the EVM.
	blockCtx := core.NewEVMBlockContext(header, processor.NewChainContext(ctx, be.Blockchain(), be.Processor().Engine), nil)
	if blockOverrides != nil {
		blockOverrides.Apply(&blockCtx)
	}
	if err := args.CallDefaults(globalGasCap, blockCtx.BaseFee, be.ChainConfig().ChainID); err != nil {
		return nil, err
	}
	msg := args.ToMessage(blockCtx.BaseFee)
	evm := vm.NewEVM(blockCtx, core.NewEVMTxContext(msg), state, be.ChainConfig(), vm.Config{NoBaseFee: true})

	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	// Execute the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.GasLimit)
	}

	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}
	return result, nil
}

func stateAndHeaderByNumberOrHash(ctx context.Context, be backend.Backend, blockNrOrHash rpc.BlockNumberOrHash) (worldstate.LayeredWorldState, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return stateAndHeaderByNumber(ctx, be, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := be.Blockchain().GetHeaderByHash(ctx, hash)
		if err != nil {
			return nil, nil, err
		}
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical {
			canonicalBlk, err := be.Blockchain().GetBlockByNumber(ctx, header.Number.Uint64())
			if err != nil {
				return nil, nil, err
			}
			if canonicalBlk.Hash() != hash {
				return nil, nil, errors.New("hash is not currently canonical")
			}
		}
		state, err := be.StateArchive().GetLayared(header.Number.Uint64(), header.Root)
		if err != nil {
			return nil, nil, err
		}
		return state, header, nil
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func stateAndHeaderByNumber(ctx context.Context, be backend.Backend, number rpc.BlockNumber) (worldstate.LayeredWorldState, *types.Header, error) {
	blk, err := getBlockByNumber(ctx, be, number)
	if err != nil {
		return nil, nil, err
	}
	header := blk.Header()

	state, err := be.StateArchive().GetLayared(header.Number.Uint64(), header.Root)
	if err != nil {
		return nil, nil, err
	}

	return state, header, nil
}
