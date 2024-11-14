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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	logging "github.com/ipfs/go-log"
	"github.com/wcgcyx/teler/blockchain"
	"github.com/wcgcyx/teler/processor"
	"github.com/wcgcyx/teler/validator"
	"github.com/wcgcyx/teler/worldstate"
)

// Logger
var log = logging.Logger("backend")

// BackendImpl implements Backend.
type BackendImpl struct {
	// Chain processor and chain config
	blkProcessor *processor.BlockProcessor
	chainConfig  *params.ChainConfig

	// Chain and states
	bc blockchain.Blockchain
	sa worldstate.LayeredWorldStateArchive

	chainHeadFeed event.Feed
	scope         event.SubscriptionScope
}

// NewBackendImpl creates a new backend.
func NewBackendImpl(
	blkProcessor *processor.BlockProcessor,
	chainConfig *params.ChainConfig,
	bc blockchain.Blockchain,
	sa worldstate.LayeredWorldStateArchive) Backend {
	return &BackendImpl{
		blkProcessor: blkProcessor,
		chainConfig:  chainConfig,
		bc:           bc,
		sa:           sa,
	}
}

// Processor gets the current block processor.
func (b *BackendImpl) Processor() *processor.BlockProcessor {
	return b.blkProcessor
}

// ChainConfig gets the current chain config.
func (b *BackendImpl) ChainConfig() *params.ChainConfig {
	return b.chainConfig
}

// Blockchain gets the blockchain instance.
func (b *BackendImpl) Blockchain() blockchain.Blockchain {
	return b.bc
}

// StateArchive gets the state archive instance.
func (b *BackendImpl) StateArchive() worldstate.LayeredWorldStateArchive {
	return b.sa
}

// ImportBlock process and import a block.
func (b *BackendImpl) ImportBlock(ctx context.Context, blk *types.Block, prvBlk *types.Block) error {
	// Check if this block has already been imported
	exists, err := b.bc.HasBlock(ctx, blk.Hash())
	if err != nil {
		return err
	}
	if exists {
		log.Infof("Block %v has already been imported, skip", blk.Hash())
		return nil
	}
	// Get previous block if not provided
	if prvBlk == nil {
		prvBlk, err = b.bc.GetBlockByHash(ctx, blk.ParentHash())
		if err != nil {
			return err
		}
	}
	// Process block
	mutableState, err := b.sa.GetMutable(prvBlk.NumberU64(), prvBlk.Header().Root)
	if err != nil {
		return err
	}
	receipts, _, gasUsed, err := b.blkProcessor.Process(ctx, blk, mutableState, vm.Config{})
	if err != nil {
		return err
	}
	err = validator.Validate(b.chainConfig, blk, receipts, gasUsed)
	if err != nil {
		return err
	}
	// Add block
	err = b.bc.AddBlock(ctx, blk, receipts)
	if err != nil {
		return err
	}
	// Finalize the state
	_, err = mutableState.Commit(blk.NumberU64(), blk.Header().Root, b.chainConfig.IsEIP158(blk.Number()))
	if err != nil && err.Error() != "already existed" {
		return err
	}
	// Set head
	err = b.bc.SetHead(ctx, blk.Hash())
	if err != nil {
		log.Warnf("Fail to set head to %v: %v", blk.Hash(), err.Error())
	}
	log.Infof("Successfully processed block %v (%v): txn %v gas used %v", blk.NumberU64(), blk.Hash(), len(receipts), gasUsed)
	// Send event
	b.chainHeadFeed.Send(core.ChainHeadEvent{Block: blk})
	return nil
}

// ImportBlocks import a list of blocks.
func (b *BackendImpl) ImportBlocks(ctx context.Context, blks []*types.Block) error {
	var prvBlk *types.Block
	for _, blk := range blks {
		err := b.ImportBlock(ctx, blk, prvBlk)
		if err != nil {
			return err
		}
		prvBlk = blk
	}
	return nil
}

// SetSafeTag sets the safe tag.
func (b *BackendImpl) SetSafeTag(ctx context.Context, safe common.Hash) {
	err := b.bc.SetSafe(ctx, safe)
	if err != nil {
		log.Warnf("Fail to set safe block to %v: %v", safe, err.Error())
	}
}

// SetFinalizedTag sets the finalized tag.
func (b *BackendImpl) SetFinalizedTag(ctx context.Context, finalized common.Hash) {
	err := b.bc.SetFinalized(ctx, finalized)
	if err != nil {
		log.Warnf("Fail to set finalized block to %v: %v", finalized, err.Error())
	}
}

// DebugDestructStateChild is used to destruct a state child.
// Note: This should ONLY be called for debug purpose.
func (b *BackendImpl) DebugDestructStateChild(height uint64, root common.Hash, child common.Hash) error {
	log.Warnf("Destruct child %v from state at %v-%v", child, height, root)
	layer, err := b.sa.GetLayared(height, root)
	if err != nil {
		return err
	}
	layer.DestructChild(child)
	return nil
}

// DebugForceProcessBlock is used to force process a block.
// Note: This should ONLY be called for debug purpose.
func (b *BackendImpl) DebugForceProcessBlock(ctx context.Context, blk *types.Block) error {
	log.Warnf("Force process block %v", blk.Hash())
	// Get prv block
	prvHash := blk.ParentHash()
	prvBlk, err := b.bc.GetBlockByHash(ctx, prvHash)
	if err != nil {
		return err
	}
	// Process block
	mutableState, err := b.sa.GetMutable(prvBlk.NumberU64(), prvBlk.Header().Root)
	if err != nil {
		return err
	}
	receipts, _, gasUsed, err := b.blkProcessor.Process(ctx, blk, mutableState, vm.Config{})
	if err != nil {
		return err
	}
	err = validator.Validate(b.chainConfig, blk, receipts, gasUsed)
	if err != nil {
		return err
	}
	// Finalize the state
	_, err = mutableState.Commit(blk.NumberU64(), blk.Header().Root, b.chainConfig.IsEIP158(blk.Number()))
	if err != nil && err.Error() != "already existed" {
		return err
	}
	log.Infof("Force processed block %v (%v): txn %v gas used %v", blk.NumberU64(), blk.Hash(), len(receipts), gasUsed)
	return nil
}

// SubscribeChainHeadEvent registers a subscription of ChainHeadEvent.
func (b *BackendImpl) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.scope.Track(b.chainHeadFeed.Subscribe(ch))
}

// Shutdown safely shuts the backend down.
func (b *BackendImpl) Shutdown() {
	log.Infof("Close backend...")
	b.scope.Close()
	log.Infof("Backend closed successfully.")
}
