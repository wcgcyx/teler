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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	logging "github.com/ipfs/go-log"
	"github.com/mus-format/mus-go/varint"
	"github.com/syndtr/goleveldb/leveldb"
	lerrors "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	itypes "github.com/wcgcyx/teler/types"
)

// Logger
var log = logging.Logger("blockchain")

// blockchainImpl implements Blockchain.
type blockchainImpl struct {
	opts Opts
	ds   *leveldb.DB

	tail uint64
}

// NewBlockchainImpl creates a new Blockchain.
func NewBlockchainImpl(_ context.Context, opts Opts, genesis *core.Genesis) (Blockchain, error) {
	if opts.Path == "" {
		return nil, fmt.Errorf("empty path provided")
	}
	ds, err := leveldb.OpenFile(opts.Path, &opt.Options{})
	if err != nil {
		return nil, err
	}
	res := &blockchainImpl{
		opts: opts,
		ds:   ds,
	}
	val, err := res.ds.Get(getTailKey(), nil)
	if err == nil {
		tail, _, err := varint.UnmarshalUint64(val)
		if err != nil {
			log.Infof("Close DB: %v", ds.Close())
			return nil, err
		}
		log.Infof("Existing ds detected, skip starting from genesis")
		res.tail = tail
		return res, nil
	}
	if !errors.Is(err, lerrors.ErrNotFound) {
		return nil, err
	}
	log.Infof("No existing ds detected, starting from genesis...")
	// Write genesis block to block, forks, head and tail.
	txn, err := res.ds.OpenTransaction()
	if err != nil {
		return nil, err
	}
	defer txn.Discard()

	genesisBlk := genesis.ToBlock()

	size := varint.SizeUint64(0)
	bs := make([]byte, size)
	varint.MarshalUint64(0, bs)
	err = txn.Put(getTailKey(), bs, nil)
	if err != nil {
		return nil, err
	}
	size = itypes.SizeHash(genesisBlk.Hash())
	bs = make([]byte, size)
	itypes.MarshalHash(genesisBlk.Hash(), bs)
	err = txn.Put(getHeadKey(), bs, nil)
	if err != nil {
		return nil, err
	}
	err = txn.Put(getForkKey(0), encodeForks([]common.Hash{genesisBlk.Hash(), genesisBlk.Hash()}), nil)
	if err != nil {
		return nil, err
	}
	err = txn.Put(getBlockKey(genesisBlk.Hash()), encodeBlock(genesisBlk), nil)
	if err != nil {
		return nil, err
	}
	err = txn.Commit()
	if err != nil {
		return nil, err
	}
	log.Infof("Datastore successfully initialized from genesis")
	res.tail = 0
	return res, nil
}

// HasBlock returns if blockchain contains the block corresponding to the block hash.
func (bc *blockchainImpl) HasBlock(_ context.Context, hash common.Hash) (bool, error) {
	return bc.ds.Has(getBlockKey(hash), nil)
}

// GetBlockByHash returns the block corresponding to the block hash.
func (bc *blockchainImpl) GetBlockByHash(_ context.Context, hash common.Hash) (*types.Block, error) {
	val, err := bc.ds.Get(getBlockKey(hash), nil)
	if err != nil {
		return nil, err
	}
	return decodeBlock(val)
}

// GetBlockByNumber returns the block corresponding to the block height.
func (bc *blockchainImpl) GetBlockByNumber(ctx context.Context, height uint64) (*types.Block, error) {
	// Get fork first
	forks, err := bc.getForks(ctx, height)
	if err != nil {
		return nil, err
	}
	if len(forks) == 0 {
		log.Errorf("Fork for %v has 0 block", height)
		return nil, fmt.Errorf("fork for %v has 0 block", height)
	}
	if forks[0].Cmp(common.Hash{}) == 0 {
		return nil, fmt.Errorf("fork for %v has no block on canonical chain", height)
	}
	return bc.GetBlockByHash(ctx, forks[0])
}

// GetHeaderByHash returns the header corresponding to the block hash.
func (bc *blockchainImpl) GetHeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	blk, err := bc.GetBlockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return blk.Header(), nil
}

// GetHeaderByNumber returns the header corresponding to the block height.
func (bc *blockchainImpl) GetHeaderByNumber(ctx context.Context, height uint64) (*types.Header, error) {
	blk, err := bc.GetBlockByNumber(ctx, height)
	if err != nil {
		return nil, err
	}
	return blk.Header(), nil
}

// GetTransaction gets the transaction for given tx hash.
func (bc *blockchainImpl) GetTransaction(_ context.Context, hash common.Hash) (*types.Transaction, common.Hash, uint64, bool, error) {
	val, err := bc.ds.Get(getTransactionKey(hash), nil)
	if err != nil {
		if errors.Is(err, lerrors.ErrNotFound) {
			return nil, common.Hash{}, 0, false, nil
		}
		return nil, common.Hash{}, 0, false, err
	}

	res, blkHash, index, err := decodeTransaction(val)
	return res, blkHash, index, true, err
}

// GetReceipt gets the receipt for given tx hash.
func (bc *blockchainImpl) GetReceipt(_ context.Context, hash common.Hash) (*types.Receipt, common.Hash, uint64, bool, error) {
	val, err := bc.ds.Get(getReceiptKey(hash), nil)
	if err != nil {
		if errors.Is(err, lerrors.ErrNotFound) {
			return nil, common.Hash{}, 0, false, nil
		}
		return nil, common.Hash{}, 0, false, err
	}

	res, blkHash, index, err := decodeReceipt(val)
	return res, blkHash, index, true, err
}

// GetHead returns the head block.
func (bc *blockchainImpl) GetHead(ctx context.Context) (*types.Block, error) {
	val, err := bc.ds.Get(getHeadKey(), nil)
	if err != nil {
		return nil, err
	}

	hash, _, err := itypes.UnmarshalHash(val)
	if err != nil {
		return nil, err
	}
	return bc.GetBlockByHash(ctx, hash)
}

// GetHead returns the finalized block.
func (bc *blockchainImpl) GetFinalized(ctx context.Context) (*types.Block, bool, error) {
	val, err := bc.ds.Get(getFinalizedKey(), nil)
	if err != nil {
		if errors.Is(err, lerrors.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}

	hash, _, err := itypes.UnmarshalHash(val)
	if err != nil {
		return nil, false, err
	}

	blk, err := bc.GetBlockByHash(ctx, hash)
	if err != nil {
		return nil, false, err
	}
	return blk, true, nil
}

// GetSafe returns the safe block.
func (bc *blockchainImpl) GetSafe(ctx context.Context) (*types.Block, bool, error) {
	val, err := bc.ds.Get(getSafeKey(), nil)
	if err != nil {
		if errors.Is(err, lerrors.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}

	hash, _, err := itypes.UnmarshalHash(val)
	if err != nil {
		return nil, false, err
	}

	blk, err := bc.GetBlockByHash(ctx, hash)
	if err != nil {
		return nil, false, err
	}
	return blk, true, nil
}

// AddBlock adds a *validated* block to the blockchain.
func (bc *blockchainImpl) AddBlock(ctx context.Context, blk *types.Block, receipts types.Receipts) error {
	parentHash := blk.ParentHash()
	hash := blk.Hash()
	if blk.NumberU64() < bc.tail {
		// Avoid adding block that's too old
		return nil
	}
	// Check if block exists
	exists, err := bc.HasBlock(ctx, hash)
	if err != nil {
		return err
	}
	if exists {
		// Avoid adding block that exists
		return nil
	}
	// Check if parent exist
	exists, err = bc.HasBlock(ctx, parentHash)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("parent %v for %v does not exist", parentHash, hash)
	}
	if len(blk.Transactions()) != len(receipts) {
		return fmt.Errorf("receipts length mismatch: expect %v, got %v", len(blk.Transactions()), len(receipts))
	}
	for i, txn := range blk.Transactions() {
		if receipts[i].TxHash.Cmp(txn.Hash()) != 0 {
			return fmt.Errorf("receipt does not match transaction, expect %v, got %v", txn.Hash(), receipts[i].TxHash)
		}
	}

	txn, err := bc.ds.OpenTransaction()
	if err != nil {
		return err
	}
	defer txn.Discard()

	// Put block
	err = txn.Put(getBlockKey(hash), encodeBlock(blk), nil)
	if err != nil {
		return err
	}

	// Update fork
	forks, err := bc.getForks(ctx, blk.NumberU64())
	if err != nil {
		if !errors.Is(err, lerrors.ErrNotFound) {
			return err
		}
		forks = make([]common.Hash, 0)
		forks = append(forks, common.Hash{})
	}
	forks = append(forks, hash)
	err = txn.Put(getForkKey(blk.NumberU64()), encodeForks(forks), nil)
	if err != nil {
		return err
	}

	// Put transaction
	for index, t := range blk.Transactions() {
		err = txn.Put(getTransactionKey(t.Hash()), encodeTransaction(t, blk.Hash(), uint64(index)), nil)
		if err != nil {
			return err
		}
	}

	// Put receipt
	for index, r := range receipts {
		err = txn.Put(getReceiptKey(r.TxHash), encodeReceipt(r, blk.Hash(), uint64(index)), nil)
		if err != nil {
			return err
		}
	}

	err = txn.Commit()
	if err != nil {
		return err
	}

	return nil
}

// tryPrune try to prune the chain by given new height.
func (bc *blockchainImpl) tryPrune(ctx context.Context, height uint64) {
	if height-bc.tail <= bc.opts.MaxBlockToRetain {
		return
	}

	numBlksPruned := 0
	numTxnsPruned := 0

	newTail := bc.tail + bc.opts.PruningFrequency
	for i := bc.tail; i < newTail; i++ {
		func() {
			txn, err := bc.ds.OpenTransaction()
			if err != nil {
				log.Warnf("Fail to start new transaction to prune: %v", err.Error())
				return
			}
			defer txn.Discard()

			// Get forks
			forks, err := bc.getForks(ctx, i)
			if err != nil {
				if !errors.Is(err, lerrors.ErrNotFound) {
					// Skip pruned forks
					return
				}
				log.Errorf("Fail to get fork data for %v: %v", i, err.Error())
				return
			}
			for _, fork := range forks {
				// Get block
				blk, err := bc.GetBlockByHash(ctx, fork)
				if err != nil {
					log.Errorf("Fail to get block %v: %v", fork, err.Error())
					return
				}
				// Delete block
				err = txn.Delete(getBlockKey(fork), nil)
				if err != nil {
					log.Errorf("Fail to remove block %v: %v", fork, err.Error())
					return
				}
				// Delete transactions & receipts
				for _, t := range blk.Transactions() {
					err = txn.Delete(getTransactionKey(t.Hash()), nil)
					if err != nil {
						log.Errorf("Fail to remove transaction %v in block %v: %v", t.Hash(), fork, err.Error())
						return
					}
					err = txn.Delete(getReceiptKey(t.Hash()), nil)
					if err != nil {
						log.Errorf("Fail to remove receipt of transaction %v in block %v: %v", t.Hash(), fork, err.Error())
						return
					}
					numTxnsPruned++
				}
				numBlksPruned++
			}
			// Delete forks
			err = txn.Delete(getForkKey(i), nil)
			if err != nil {
				log.Errorf("Fail to delete fork data for %v: %v", i, err.Error())
				return
			}
			err = txn.Commit()
			if err != nil {
				log.Errorf("Fail to commit to prune blocks at %v: %v", i, err.Error())
				return
			}
		}()
	}

	// Update new tail
	size := varint.SizeUint64(newTail)
	bs := make([]byte, size)
	varint.MarshalUint64(newTail, bs)
	err := bc.ds.Put(getTailKey(), bs, nil)
	if err != nil {
		log.Errorf("Fail to update tail from %v to %v: %v", bc.tail, newTail, err.Error())
		return
	}
	log.Infof("Succesfully pruned block data from %v to %v: %v blks pruned, %v txns pruned", bc.tail, newTail, numBlksPruned, numTxnsPruned)

	bc.tail = newTail
}

// SetHead sets the head of the chain.
func (bc *blockchainImpl) SetHead(ctx context.Context, hash common.Hash) error {
	// Check if this block exists
	blk, err := bc.GetBlockByHash(ctx, hash)
	if err != nil {
		return err
	}

	txn, err := bc.ds.OpenTransaction()
	if err != nil {
		return err
	}
	defer txn.Discard()

	// Get the current head
	val, err := bc.ds.Get(getHeadKey(), nil)
	if err != nil {
		// This should never happen.
		return err
	}

	// Get current head
	currentHeadHash, _, err := itypes.UnmarshalHash(val)
	if err != nil {
		return err
	}
	currentHead, err := bc.GetBlockByHash(ctx, currentHeadHash)
	if err != nil {
		return err
	}
	if currentHead.NumberU64() > blk.NumberU64() {
		log.Warnf("Current head %v is higher than new head %v, likely a reorg occurred.", currentHead.NumberU64(), blk.NumberU64())
		// Clear canonical chain from current head to blk number + 1.
		for i := blk.NumberU64() + 1; i <= currentHead.NumberU64(); i++ {
			// Get forks
			forks, err := bc.getForks(ctx, i)
			if err != nil {
				log.Errorf("Fail to get fork data for %v: %v", i, err.Error())
				return err
			}
			forks[0] = common.Hash{}
			err = txn.Put(getForkKey(i), encodeForks(forks), nil)
			if err != nil {
				return err
			}
		}
	}
	// Get forks for new head
	forks, err := bc.getForks(ctx, blk.NumberU64())
	if err != nil {
		log.Errorf("Fail to get fork data for %v: %v", blk.NumberU64(), err.Error())
		return err
	}
	forks[0] = blk.Hash()
	err = txn.Put(getForkKey(blk.NumberU64()), encodeForks(forks), nil)
	if err != nil {
		return err
	}
	// Set canonical chain from new head to a node that's on canonical chain.
	parentBlkHash := blk.ParentHash()
	parentBlk, err := bc.GetBlockByHash(ctx, parentBlkHash)
	if err != nil {
		return err
	}
	parentHeight := blk.NumberU64() - 1
	if parentHeight < bc.tail {
		return fmt.Errorf("parent height %v is lower than blockchain tail %v, a major reorg occured", parentHeight, bc.tail)
	}
	parentForks, err := bc.getForks(ctx, parentHeight)
	if err != nil {
		log.Errorf("Fail to get fork data for %v: %v", parentHeight, err.Error())
		return err
	}
	for parentForks[0] != parentBlkHash {
		if parentForks[0].Cmp(common.Hash{}) != 0 {
			log.Infof("Canonical chain at height %v reorg from %v to %v", parentHeight, parentForks[0], parentBlkHash)
		}
		parentForks[0] = parentBlkHash
		err = txn.Put(getForkKey(parentHeight), encodeForks(parentForks), nil)
		if err != nil {
			return err
		}
		// Get grandparent
		grandparentBlkHash := parentBlk.ParentHash()
		grandparentBlk, err := bc.GetBlockByHash(ctx, grandparentBlkHash)
		if err != nil {
			return err
		}
		grandparentHeight := parentHeight - 1
		if grandparentHeight < bc.tail {
			return fmt.Errorf("grandparent height %v is lower than blockchain tail %v, a major reorg occured", grandparentHeight, bc.tail)
		}
		grandparentForks, err := bc.getForks(ctx, grandparentHeight)
		if err != nil {
			log.Errorf("Fail to get fork data for %v: %v", grandparentHeight, err.Error())
			return err
		}
		parentBlkHash = grandparentBlkHash
		parentBlk = grandparentBlk
		parentHeight = grandparentHeight
		parentForks = grandparentForks
	}
	// Set head to be the new head
	size := itypes.SizeHash(hash)
	bs := make([]byte, size)
	itypes.MarshalHash(hash, bs)
	err = txn.Put(getHeadKey(), bs, nil)
	if err != nil {
		return err
	}
	err = txn.Commit()
	if err == nil {
		// Try to prune.
		bc.tryPrune(ctx, blk.NumberU64())
	}
	return err
}

// SetFinalized sets the finalized block of this chain.
func (bc *blockchainImpl) SetFinalized(_ context.Context, hash common.Hash) error {
	size := itypes.SizeHash(hash)
	bs := make([]byte, size)
	itypes.MarshalHash(hash, bs)

	return bc.ds.Put(getFinalizedKey(), bs, nil)
}

// SetSafe sets the safe block of this chain.
func (bc *blockchainImpl) SetSafe(_ context.Context, hash common.Hash) error {
	size := itypes.SizeHash(hash)
	bs := make([]byte, size)
	itypes.MarshalHash(hash, bs)

	return bc.ds.Put(getSafeKey(), bs, nil)
}

// getForks gets the forks for given height.
func (bc *blockchainImpl) getForks(_ context.Context, height uint64) ([]common.Hash, error) {
	// Get forks
	forkVal, err := bc.ds.Get(getForkKey(height), nil)
	if err != nil {
		return nil, err
	}
	return decodeForks(forkVal)
}

// Shutdown safely shuts the blockchain down.
func (bc *blockchainImpl) Shutdown() {
	log.Infof("Close blockchain...")
	err := bc.ds.Close()
	if err != nil {
		log.Errorf("Fail to close blockchain: %v", err.Error())
		return
	}
	log.Infof("Blockchain closed successfully.")
}
