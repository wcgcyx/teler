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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/wcgcyx/teler/backend"
	"github.com/wcgcyx/teler/node"
)

// Note:
// This is adapted from:
// 		go-ethereum@v1.14.8/internal/ethapi/api.go

// adminAPIHandler is used to handle admin API.
type adminAPIHandler struct {
	opts Opts

	node *node.Node
	be   backend.Backend
}

func (h *adminAPIHandler) Pause() error {
	h.node.Pause()
	return nil
}

func (h *adminAPIHandler) Unpause() error {
	h.node.Unpause()
	return nil
}

func (h *adminAPIHandler) DestructStateChild(height uint64, root common.Hash, child common.Hash) error {
	return h.be.DebugDestructStateChild(height, root, child)
}

func (h *adminAPIHandler) ForceProcessBlock(ctx context.Context, hash common.Hash) error {
	exists, err := h.be.Blockchain().HasBlock(ctx, hash)
	if err != nil {
		return err
	}
	var blk *types.Block
	if exists {
		blk, err = h.be.Blockchain().GetBlockByHash(ctx, hash)
	} else {
		blk, err = h.node.BlkSrc.BlockByHash(ctx, hash)
	}
	if err != nil {
		return err
	}
	return h.be.DebugForceProcessBlock(ctx, blk)
}

// ethAPIHandler is used to handle eth API.
type ethAPIHandler struct {
	opts Opts

	be backend.Backend
}

func (h *ethAPIHandler) BlobBaseFee(ctx context.Context) (*big.Int, error) {
	blk, err := h.be.Blockchain().GetHead(ctx)
	if err != nil {
		return nil, err
	}
	if excess := blk.ExcessBlobGas(); excess != nil {
		return eip4844.CalcBlobFee(*excess), nil
	}
	return nil, nil
}

func (h *ethAPIHandler) BlockNumber(ctx context.Context) (*big.Int, error) {
	blk, err := h.be.Blockchain().GetHead(ctx)
	if err != nil {
		return nil, err
	}
	return blk.Number(), nil
}

func (h *ethAPIHandler) Call(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, overrides *StateOverride, blockOverrides *BlockOverrides) (hexutil.Bytes, error) {
	if blockNrOrHash == nil {
		latest := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
		blockNrOrHash = &latest
	}
	result, err := DoCall(ctx, h.be, args, *blockNrOrHash, overrides, blockOverrides, h.opts.RPCEVMTimeout, h.opts.RPCGasCap)
	if err != nil {
		return nil, err
	}
	// If the result contains a revert reason, try to unpack and return it.
	if len(result.Revert()) > 0 {
		return nil, newRevertError(result.Revert())
	}
	return result.Return(), result.Err
}

func (h *ethAPIHandler) ChainId() *big.Int {
	return h.be.ChainConfig().ChainID
}

func (h *ethAPIHandler) EstimateGas(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, overrides *StateOverride) (hexutil.Uint64, error) {
	bNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}
	return DoEstimateGas(ctx, h.be, args, bNrOrHash, overrides, h.opts.RPCGasCap)
}

func (h *ethAPIHandler) GetBalance(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	layer, _, err := stateAndHeaderByNumberOrHash(ctx, h.be, blockNrOrHash)
	if layer == nil || err != nil {
		return nil, err
	}
	b := layer.GetReadOnly().GetBalance(address).ToBig()
	return (*hexutil.Big)(b), nil
}

func (h *ethAPIHandler) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block, err := h.be.Blockchain().GetBlockByHash(ctx, hash)
	if block != nil {
		return rpcMarshalBlock(h.be, block, true, fullTx)
	}
	return nil, err
}

func (h *ethAPIHandler) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	block, err := getBlockByNumber(ctx, h.be, number)
	if block != nil && err == nil {
		response, err := rpcMarshalBlock(h.be, block, true, fullTx)
		if err == nil && number == rpc.PendingBlockNumber {
			// Pending blocks need to nil out a few fields
			for _, field := range []string{"hash", "nonce", "miner"} {
				response[field] = nil
			}
		}
		return response, err
	}
	return nil, err
}

func (h *ethAPIHandler) GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]map[string]interface{}, error) {
	var block *types.Block
	var err error
	if blockNr, ok := blockNrOrHash.Number(); ok {
		block, err = getBlockByNumber(ctx, h.be, blockNr)
	} else if hash, ok := blockNrOrHash.Hash(); ok {
		block, err = h.be.Blockchain().GetBlockByHash(ctx, hash)
	}
	if block == nil || err != nil {
		// When the block doesn't exist, the RPC method should return JSON null as per specification.
		return nil, nil
	}

	receipts := make([]*types.Receipt, 0)
	for _, tx := range block.Transactions() {
		receipt, _, _, exists, err := h.be.Blockchain().GetReceipt(ctx, tx.Hash())
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("receipt for tx %v does not exist", tx.Hash())
		}
		receipts = append(receipts, receipt)
	}

	txs := block.Transactions()
	if len(txs) != len(receipts) {
		return nil, fmt.Errorf("receipts length mismatch: %d vs %d", len(txs), len(receipts))
	}

	// Derive the sender.
	signer := types.MakeSigner(h.be.ChainConfig(), block.Number(), block.Time())

	result := make([]map[string]interface{}, len(receipts))
	for i, receipt := range receipts {
		result[i] = marshalReceipt(receipt, block.Hash(), block.NumberU64(), signer, txs[i], i)
	}

	return result, nil
}

func (h *ethAPIHandler) GetBlockTransactionCountByHash(ctx context.Context, blockHash common.Hash) *hexutil.Uint {
	block, err := h.be.Blockchain().GetBlockByHash(ctx, blockHash)
	if err != nil {
		return nil
	}
	res := hexutil.Uint(block.Transactions().Len())
	return &res
}

func (h *ethAPIHandler) GetBlockTransactionCountByNumber(ctx context.Context, blockNr rpc.BlockNumber) *hexutil.Uint {
	block, err := getBlockByNumber(ctx, h.be, blockNr)
	if err != nil {
		return nil
	}
	res := hexutil.Uint(block.Transactions().Len())
	return &res
}

func (h *ethAPIHandler) GetCode(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	layer, _, err := stateAndHeaderByNumberOrHash(ctx, h.be, blockNrOrHash)
	if layer == nil || err != nil {
		return nil, err
	}
	return layer.GetReadOnly().GetCode(address), nil
}

func (h *ethAPIHandler) GetStorageAt(ctx context.Context, address common.Address, hexKey string, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	layer, _, err := stateAndHeaderByNumberOrHash(ctx, h.be, blockNrOrHash)
	if layer == nil || err != nil {
		return nil, err
	}
	key, _, err := decodeHash(hexKey)
	if err != nil {
		return nil, fmt.Errorf("unable to decode storage key: %s", err)
	}
	res := layer.GetReadOnly().GetState(address, key)
	return res[:], nil
}

func (h *ethAPIHandler) GetTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) *RPCTransaction {
	block, err := h.be.Blockchain().GetBlockByHash(ctx, blockHash)
	if err != nil {
		return nil
	}
	return newRPCTransactionFromBlockIndex(block, uint64(index), h.be.ChainConfig())
}

func (h *ethAPIHandler) GetTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) *RPCTransaction {
	block, err := getBlockByNumber(ctx, h.be, blockNr)
	if err != nil {
		return nil
	}
	return newRPCTransactionFromBlockIndex(block, uint64(index), h.be.ChainConfig())
}

func (h *ethAPIHandler) GetTransactionByHash(ctx context.Context, hash common.Hash) (*RPCTransaction, error) {
	tx, blkHash, index, exists, err := h.be.Blockchain().GetTransaction(ctx, hash)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NewTxIndexingError()
	}
	header, err := h.be.Blockchain().GetHeaderByHash(ctx, blkHash)
	if err != nil {
		return nil, err
	}
	return newRPCTransaction(tx, blkHash, header.Number.Uint64(), header.Time, index, header.BaseFee, h.be.ChainConfig()), nil
}

func (h *ethAPIHandler) GetTransactionCount(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Uint64, error) {
	layer, _, err := stateAndHeaderByNumberOrHash(ctx, h.be, blockNrOrHash)
	if layer == nil || err != nil {
		return nil, err
	}
	nonce := layer.GetReadOnly().GetNonce(address)
	return (*hexutil.Uint64)(&nonce), nil
}

func (h *ethAPIHandler) GetTransactionReceipt(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	tx, _, _, exists, err := h.be.Blockchain().GetTransaction(ctx, hash)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NewTxIndexingError()
	}
	receipt, blkHash, index, exists, err := h.be.Blockchain().GetReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NewTxIndexingError()
	}
	header, err := h.be.Blockchain().GetHeaderByHash(ctx, blkHash)
	if err != nil {
		return nil, err
	}
	// Derive the sender.
	signer := types.MakeSigner(h.be.ChainConfig(), header.Number, header.Time)
	return marshalReceipt(receipt, blkHash, header.Number.Uint64(), signer, tx, int(index)), nil
}

func (h *ethAPIHandler) GetUncleByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := getBlockByNumber(ctx, h.be, blockNr)
	if err != nil {
		return nil, err
	}
	uncles := block.Uncles()
	if index >= hexutil.Uint(len(uncles)) {
		return nil, nil
	}
	block = types.NewBlockWithHeader(uncles[index])
	return rpcMarshalBlock(h.be, block, false, false)
}

func (h *ethAPIHandler) GetUncleByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := h.be.Blockchain().GetBlockByHash(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	uncles := block.Uncles()
	if index >= hexutil.Uint(len(uncles)) {
		return nil, nil
	}
	block = types.NewBlockWithHeader(uncles[index])
	return rpcMarshalBlock(h.be, block, false, false)
}
