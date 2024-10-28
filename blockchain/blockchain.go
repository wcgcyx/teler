package blockchain

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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Blockchain interface {
	// HasBlock returns if blockchain contains the block corresponding to the block hash.
	HasBlock(context.Context, common.Hash) (bool, error)

	// GetBlockByHash returns the block corresponding to the block hash.
	GetBlockByHash(context.Context, common.Hash) (*types.Block, error)

	// GetBlockByNumber returns the block corresponding to the block height.
	GetBlockByNumber(context.Context, uint64) (*types.Block, error)

	// GetHeaderByHash returns the header corresponding to the block hash.
	GetHeaderByHash(context.Context, common.Hash) (*types.Header, error)

	// GetHeaderByNumber returns the header corresponding to the block height.
	GetHeaderByNumber(context.Context, uint64) (*types.Header, error)

	// GetTransaction gets the transaction for given tx hash.
	GetTransaction(context.Context, common.Hash) (*types.Transaction, common.Hash, uint64, bool, error)

	// GetReceipt gets the receipt for given tx hash.
	GetReceipt(context.Context, common.Hash) (*types.Receipt, common.Hash, uint64, bool, error)

	// GetHead returns the head block.
	GetHead(context.Context) (*types.Block, error)

	// GetHead returns the finalized block.
	GetFinalized(context.Context) (*types.Block, bool, error)

	// GetSafe returns the safe block.
	GetSafe(context.Context) (*types.Block, bool, error)

	// AddBlock adds a *validated* block to the blockchain.
	AddBlock(context.Context, *types.Block, types.Receipts) error

	// SetHead sets the head of the chain.
	SetHead(context.Context, common.Hash) error

	// SetFinalized sets the finalized block of this chain.
	SetFinalized(context.Context, common.Hash) error

	// SetSafe sets the safe block of this chain.
	SetSafe(context.Context, common.Hash) error

	// Shutdown safely shuts the blockchain down.
	Shutdown()
}
