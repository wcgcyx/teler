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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/wcgcyx/teler/backend"
)

// Note:
// This is adapted from:
// 		go-ethereum@v1.15.6/internal/ethapi/api.go
// 		go-ethereum@v1.15.6/eth/gasprice/gasprice.go

type feeHistoryResult struct {
	OldestBlock      *hexutil.Big     `json:"oldestBlock"`
	Reward           [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee          []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio     []float64        `json:"gasUsedRatio"`
	BlobBaseFee      []*hexutil.Big   `json:"baseFeePerBlobGas,omitempty"`
	BlobGasUsedRatio []float64        `json:"blobGasUsedRatio,omitempty"`
}

type compatibleOracleBackend struct {
	be backend.Backend
}

func (ob *compatibleOracleBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	block, err := getBlockByNumber(ctx, ob.be, number)
	if err != nil {
		return nil, err
	}
	return block.Header(), nil
}

func (ob *compatibleOracleBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	return getBlockByNumber(ctx, ob.be, number)
}

func (ob *compatibleOracleBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	block, err := ob.be.Blockchain().GetBlockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	receipts := make([]*types.Receipt, 0)
	for _, tx := range block.Transactions() {
		receipt, _, _, exists, err := ob.be.Blockchain().GetReceipt(ctx, tx.Hash())
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("receipt for tx %v does not exist", tx.Hash())
		}
		receipts = append(receipts, receipt)
	}
	return receipts, nil
}

func (ob *compatibleOracleBackend) Pending() (*types.Block, types.Receipts, *state.StateDB) {
	return nil, nil, nil
}

func (ob *compatibleOracleBackend) ChainConfig() *params.ChainConfig {
	return ob.be.ChainConfig()
}

func (ob *compatibleOracleBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return ob.be.SubscribeChainHeadEvent(ch)
}
