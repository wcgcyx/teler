package sync

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

	"github.com/ethereum/go-ethereum/core/types"
	logging "github.com/ipfs/go-log"
	"github.com/wcgcyx/teler/backend"
)

// Logger
var log = logging.Logger("sync")

// ForwardSync syncs from current head to target number.
func ForwardSync(ctx context.Context, b backend.Backend, blkSrc BlockSource, target uint64) error {
	// Get current head
	prvBlk, err := b.Blockchain().GetHead(ctx)
	if err != nil {
		return err
	}
	log.Infof("Start forward sync from %v to %v", prvBlk.NumberU64(), target)

	// Use a separate go routine to pull blocks
	blkChan := make(chan *types.Block, 100)
	errChan := make(chan error, 1)
	defer close(errChan)

	go func(start uint64, target uint64) {
		defer close(blkChan)
		for i := start; i <= target; i++ {
			select {
			case <-ctx.Done():
				log.Warnf("Sync cancelled while fetching blocks, current height %v", i)
				return
			default:
				blk, err := blkSrc.BlockByNumber(ctx, big.NewInt(int64(i)))
				if err != nil {
					log.Errorf("Failed to fetch block %v: %v", i, err)
					errChan <- err
					return
				}
				select {
				case blkChan <- blk:
				case <-ctx.Done():
					return
				}
			}
		}
	}(prvBlk.NumberU64()+1, target)

	// Process blocks
	for prvBlk.NumberU64() < target {
		select {
		case err := <-errChan:
			return err
		case <-ctx.Done():
			log.Warnf("Sync cancelled while importing blocks, last height %v", prvBlk.NumberU64())
			return ctx.Err()
		case blk, ok := <-blkChan:
			if !ok {
				return nil
			}
			err = b.ImportBlock(ctx, blk, prvBlk)
			if err != nil {
				log.Errorf("Fail to import block %v-%v: %v", blk.NumberU64(), blk.Hash(), err.Error())
				return err
			}
			prvBlk = blk
		}
	}

	return nil
}

// BackwardSync syncs from target backward to current head.
func BackwardSync(ctx context.Context, b backend.Backend, blkSrc BlockSource, target *types.Block, maxBlksToQuery uint64) error {
	blocks := make([]*types.Block, 0)
	blocks = append([]*types.Block{target}, blocks...)
	parent := target.ParentHash()
	exists, err := b.Blockchain().HasBlock(ctx, parent)
	if err != nil {
		return err
	}
	log.Infof("Start backward sync from %v (%v) with max %v blocks to query", target.NumberU64(), target.Hash(), maxBlksToQuery)
	// Keep querying until canonical chain is reached
	attempts := uint64(0)
	for !exists {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if attempts >= maxBlksToQuery {
			return fmt.Errorf("fail to reach canonical chain from %v with %v blks queried", target.Hash(), maxBlksToQuery)
		}
		attempts++
		blk, err := blkSrc.BlockByHash(ctx, parent)
		if err != nil {
			return err
		}
		blocks = append([]*types.Block{blk}, blocks...)
		parent = blk.Header().ParentHash
		exists, err = b.Blockchain().HasBlock(ctx, parent)
		if err != nil {
			return err
		}
	}
	log.Infof("Backward sync start importing %v blocks", len(blocks))
	err = b.ImportBlocks(ctx, blocks)
	if err != nil {
		log.Errorf("Fail to import blocks: %v", err.Error())
		return err
	}
	return nil
}
