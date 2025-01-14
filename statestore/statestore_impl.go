package statestore

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

	"github.com/dgraph-io/badger/v2/options"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/ipfs/go-datastore"
	badgerds "github.com/ipfs/go-ds-badger2"
	logging "github.com/ipfs/go-log"
	itypes "github.com/wcgcyx/teler/types"
)

// Logger
var log = logging.Logger("statestore")

// stateStoreImpl implements StateStore.
type stateStoreImpl struct {
	ctx  context.Context
	opts Opts
	ds   *badgerds.Datastore
	// Process related
	routineCtx context.Context
	cancel     context.CancelFunc
	exitLoop   chan bool
}

// NewStateStoreImpl creates a new StateStore
func NewStateStoreImpl(ctx context.Context, opts Opts, genesis *core.Genesis, genesisRoot common.Hash) (StateStore, error) {
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	// Use max table size of 256MiB
	dsopts.Options.MaxTableSize = 256 << 20
	// Use memory map for value log
	dsopts.Options.ValueLogLoadingMode = options.MemoryMap
	if opts.Path == "" {
		return nil, fmt.Errorf("empty path provided")
	}
	ds, err := badgerds.NewDatastore(opts.Path, &dsopts)
	if err != nil {
		return nil, err
	}
	routineCtx, cancel := context.WithCancel(context.Background())
	res := &stateStoreImpl{
		ctx:        ctx,
		opts:       opts,
		ds:         ds,
		routineCtx: routineCtx,
		cancel:     cancel,
		exitLoop:   make(chan bool),
	}
	ok, err := res.ds.Has(ctx, persistedHeightKey())
	if err != nil {
		ds.Close()
		return nil, err
	}
	if ok {
		log.Infof("Existing ds detected, skip starting from genesis")
		go res.gcRoutine()
		return res, nil
	}
	if genesis == nil {
		ds.Close()
		return nil, fmt.Errorf("empty genesis provided")
	}
	// Persist genesis state
	layerLog := itypes.LayerLogFromGenesis(genesis, genesisRoot)
	txn, err := res.NewTransaction()
	defer txn.Discard()
	if err != nil {
		ds.Close()
		return nil, err
	}
	defer txn.Discard()
	err = txn.PersistLayerLog(*layerLog)
	if err != nil {
		ds.Close()
		return nil, err
	}
	err = txn.PutChildren(layerLog.BlockNumber, layerLog.RootHash, []common.Hash{})
	if err != nil {
		ds.Close()
		return nil, err
	}
	err = txn.Commit()
	if err != nil {
		ds.Close()
		return nil, err
	}
	go res.gcRoutine()
	return res, nil
}

// GetPersistedHeight gets the persisted state height and root.
func (s *stateStoreImpl) GetPersistedHeight() (uint64, common.Hash, error) {
	ctx, cancel := context.WithTimeout(s.ctx, s.opts.ReadTimeout)
	defer cancel()

	val, err := s.ds.Get(ctx, persistedHeightKey())
	if err != nil {
		return 0, common.Hash{}, err
	}

	return decodePersistedHeight(val)
}

// GetChildren gets the child states for a state with the given root hash.
func (s *stateStoreImpl) GetChildren(height uint64, rootHash common.Hash) ([]common.Hash, error) {
	ctx, cancel := context.WithTimeout(s.ctx, s.opts.ReadTimeout)
	defer cancel()

	val, err := s.ds.Get(ctx, getChildrenKey(height, rootHash))
	if err != nil {
		return nil, err
	}

	return decodeChildren(val)
}

// GetLayerLog gets the layer log for a state with the given root hash.
func (s *stateStoreImpl) GetLayerLog(height uint64, rootHash common.Hash) (itypes.LayerLog, error) {
	ctx, cancel := context.WithTimeout(s.ctx, s.opts.ReadTimeout)
	defer cancel()

	val, err := s.ds.Get(ctx, getLayerLogKey(height, rootHash))
	if err != nil {
		return itypes.LayerLog{}, err
	}

	return decodeLayerLog(val)
}

// GetAccountValue gets the persisted account value for given address.
func (s *stateStoreImpl) GetAccountValue(addr common.Address) (itypes.AccountValue, error) {
	ctx, cancel := context.WithTimeout(s.ctx, s.opts.ReadTimeout)
	defer cancel()

	res := itypes.AccountValue{}
	val, err := s.ds.Get(ctx, getAccountValueKey(addr))
	if err != nil {
		if !errors.Is(err, datastore.ErrNotFound) {
			return itypes.AccountValue{}, err
		}
		log.Debugf("Get account value empty for %v", addr)
		res.Nonce = 0
		res.Balance = uint256.NewInt(0)
		res.CodeHash = types.EmptyCodeHash
		res.DirtyStorage = false
		// Get account version
		version := uint64(0)
		versionBytes, err := s.ds.Get(ctx, getAccountVersionKey(addr))
		if err == nil {
			version = decodeAccountVersion(versionBytes)
		} else if !errors.Is(err, datastore.ErrNotFound) {
			return itypes.AccountValue{}, err
		}
		res.Version = version
	} else {
		log.Debugf("Get account value non-empty for %v", addr)
		raw, err := decodeAccountValue(val)
		if err != nil {
			return itypes.AccountValue{}, err
		}
		res.Nonce = raw.Nonce
		res.Balance = raw.Balance
		res.CodeHash = raw.CodeHash
		res.DirtyStorage = raw.DirtyStorage
		res.Version = raw.Version
	}
	return res, nil
}

// GetStorageByVersion gets the persisted storage value for given key.
func (s *stateStoreImpl) GetStorageByVersion(addr common.Address, version uint64, key common.Hash) (common.Hash, error) {
	ctx, cancel := context.WithTimeout(s.ctx, s.opts.ReadTimeout)
	defer cancel()

	val, err := s.ds.Get(ctx, getStorageKey(addr, version, key))
	if err != nil {
		if errors.Is(err, datastore.ErrNotFound) {
			log.Debugf("Get storage empty for %v-%v-%v", addr, version, key)
			return common.Hash{}, nil
		}
		return common.Hash{}, err
	}

	log.Debugf("Get storage non-empty for %v-%v-%v", addr, version, key)
	return decodeStorage(val)
}

// GetCodeByHash gets the persisted code for given hash.
func (s *stateStoreImpl) GetCodeByHash(codeHash common.Hash) ([]byte, error) {
	ctx, cancel := context.WithTimeout(s.ctx, s.opts.ReadTimeout)
	defer cancel()

	codeBytes, err := s.ds.Get(ctx, getCodeKey(codeHash))
	if err != nil {
		return nil, err
	}

	code, _, err := decodeCodeValue(codeBytes)
	if err != nil {
		return nil, err
	}

	return code, nil
}

// Shutdown safely shuts the statestore down.
func (s *stateStoreImpl) Shutdown() {
	log.Infof("Close statestore...")
	s.cancel()
	<-s.exitLoop
	err := s.ds.Close()
	if err != nil {
		log.Errorf("Fail to close statestore: %v", err.Error())
		return
	}
	log.Infof("Statestore closed successfully.")
}
