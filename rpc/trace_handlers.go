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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/wcgcyx/teler/backend"
	"github.com/wcgcyx/teler/processor"
	"github.com/wcgcyx/teler/worldstate"
)

// Note:
// This is adapted from:
// 		go-ethereum@v1.15.6/eth/tracers/api.go

// traceAPIHandler is used to handle trace API.
type traceAPIHandler struct {
	opts Opts

	be backend.Backend
}

func (h *traceAPIHandler) BlockByNumber(ctx context.Context, number rpc.BlockNumber, config *tracers.TraceConfig) ([]*txTraceResult, error) {
	block, err := getBlockByNumber(ctx, h.be, number)
	if err != nil {
		return nil, err
	}
	return traceBlock(ctx, h.be, block, config)
}

func (h *traceAPIHandler) BlockByHash(ctx context.Context, hash common.Hash, config *tracers.TraceConfig) ([]*txTraceResult, error) {
	block, err := h.be.Blockchain().GetBlockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return traceBlock(ctx, h.be, block, config)
}

func (h *traceAPIHandler) Transaction(ctx context.Context, hash common.Hash, config *tracers.TraceConfig) (interface{}, error) {
	_, blockHash, index, found, err := h.be.Blockchain().GetTransaction(ctx, hash)
	if err != nil {
		return nil, NewTxIndexingError()
	}
	if !found {
		return nil, errors.New("transaction not found")
	}
	reexec := defaultTraceReexec
	if config != nil && config.Reexec != nil {
		reexec = *config.Reexec
	}
	block, err := h.be.Blockchain().GetBlockByHash(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	tx, vmctx, statedb, release, err := stateAtTransaction(ctx, h.be, block, int(index), reexec)
	if err != nil {
		return nil, err
	}
	defer release()
	msg, err := core.TransactionToMessage(tx, types.MakeSigner(h.be.ChainConfig(), block.Number(), block.Time()), block.BaseFee())
	if err != nil {
		return nil, err
	}

	txctx := &tracers.Context{
		BlockHash:   blockHash,
		BlockNumber: block.Number(),
		TxIndex:     int(index),
		TxHash:      hash,
	}
	return traceTx(ctx, h.be, tx, msg, txctx, vmctx, statedb, config)
}

func (h *traceAPIHandler) Call(ctx context.Context, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, config *TraceCallConfig) (interface{}, error) {
	// Try to retrieve the specified block
	var (
		err     error
		block   *types.Block
		state   worldstate.MutableWorldState
		release tracers.StateReleaseFunc
	)
	if hash, ok := blockNrOrHash.Hash(); ok {
		block, err = h.be.Blockchain().GetBlockByHash(ctx, hash)
	} else if number, ok := blockNrOrHash.Number(); ok {
		if number == rpc.PendingBlockNumber {
			// We don't have access to the miner here. For tracing 'future' transactions,
			// it can be done with block- and state-overrides instead, which offers
			// more flexibility and stability than trying to trace on 'pending', since
			// the contents of 'pending' is unstable and probably not a true representation
			// of what the next actual block is likely to contain.
			return nil, errors.New("tracing on top of pending is not supported")
		}
		block, err = getBlockByNumber(ctx, h.be, number)
	} else {
		return nil, errors.New("invalid arguments; neither block nor hash specified")
	}
	if err != nil {
		return nil, err
	}
	// try to recompute the state
	reexec := defaultTraceReexec
	if config != nil && config.Reexec != nil {
		reexec = *config.Reexec
	}

	if config != nil && config.TxIndex != nil {
		_, _, state, release, err = stateAtTransaction(ctx, h.be, block, int(*config.TxIndex), reexec)
	} else {
		state, err = h.be.StateArchive().GetMutable(block.NumberU64(), block.Root())
		release = func() {}
	}
	if err != nil {
		return nil, err
	}
	defer release()

	vmctx := core.NewEVMBlockContext(block.Header(), processor.NewChainContext(ctx, h.be.Blockchain(), h.be.Processor().Engine, h.be.ChainConfig()), nil)
	// Apply the customization rules if required.
	if config != nil {
		if err := config.StateOverrides.Apply(state); err != nil {
			return nil, err
		}
		config.BlockOverrides.Apply(&vmctx)
	}
	// Execute the trace
	if err := args.CallDefaults(h.opts.RPCGasCap, vmctx.BaseFee, h.be.ChainConfig().ChainID); err != nil {
		return nil, err
	}
	var (
		msg         = args.ToMessage(vmctx.BaseFee, true, true)
		tx          = args.ToTransaction()
		traceConfig *tracers.TraceConfig
	)
	if config != nil {
		traceConfig = &config.TraceConfig
	}
	return traceTx(ctx, h.be, tx, msg, new(tracers.Context), vmctx, state, traceConfig)
}
