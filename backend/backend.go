package backend

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
	"github.com/ethereum/go-ethereum/params"
	"github.com/wcgcyx/teler/blockchain"
	"github.com/wcgcyx/teler/processor"
	"github.com/wcgcyx/teler/worldstate"
)

type Backend interface {
	// Processor gets the current block processor.
	Processor() *processor.BlockProcessor

	// ChainConfig gets the current chain config.
	ChainConfig() *params.ChainConfig

	// Blockchain gets the blockchain instance.
	Blockchain() blockchain.Blockchain

	// StateArchive gets the state archive instance.
	StateArchive() worldstate.LayeredWorldStateArchive

	// ImportBlock process and import a block.
	ImportBlock(ctx context.Context, blk *types.Block, prvBlk *types.Block) error

	// ImportBlocks import a list of blocks.
	ImportBlocks(ctx context.Context, blks []*types.Block) error

	// SetSafeTag sets the safe tag.
	SetSafeTag(ctx context.Context, safe common.Hash)

	// SetFinalizedTag sets the finalized tag.
	SetFinalizedTag(ctx context.Context, finalized common.Hash)

	// DebugDestructStateChild is used to destruct a state child.
	// Note: This should ONLY be called for debug purpose.
	DebugDestructStateChild(height uint64, root common.Hash, child common.Hash) error

	// DebugForceProcessBlock is used to force process a block.
	// Note: This should ONLY be called for debug purpose.
	DebugForceProcessBlock(ctx context.Context, blk *types.Block) error
}
