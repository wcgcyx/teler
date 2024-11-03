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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	logging "github.com/ipfs/go-log"
	"github.com/syndtr/goleveldb/leveldb"
	lerrors "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	itypes "github.com/wcgcyx/teler/types"
)

// Logger
var log = logging.Logger("statestore")

// stateStoreImpl implements StateStore.
type stateStoreImpl struct {
	ctx  context.Context
	opts Opts
	ds   *leveldb.DB
	// Process related
	routineCtx context.Context
	cancel     context.CancelFunc
	exitLoop   chan bool
}

// NewStateStoreImpl creates a new StateStore
func NewStateStoreImpl(ctx context.Context, opts Opts, genesis *core.Genesis, genesisRoot common.Hash) (StateStore, error) {
	if opts.Path == "" {
		return nil, fmt.Errorf("empty path provided")
	}
	ds, err := leveldb.OpenFile(opts.Path, &opt.Options{})
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
	ok, err := res.ds.Has(persistedHeightKey(), nil)
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
	val, err := s.ds.Get(persistedHeightKey(), nil)
	if err != nil {
		return 0, common.Hash{}, err
	}

	return decodePersistedHeight(val)
}

// GetChildren gets the child states for a state with the given root hash.
func (s *stateStoreImpl) GetChildren(height uint64, rootHash common.Hash) ([]common.Hash, error) {
	val, err := s.ds.Get(getChildrenKey(height, rootHash), nil)
	if err != nil {
		return nil, err
	}

	return decodeChildren(val)
}

// GetLayerLog gets the layer log for a state with the given root hash.
func (s *stateStoreImpl) GetLayerLog(height uint64, rootHash common.Hash) (itypes.LayerLog, error) {
	val, err := s.ds.Get(getLayerLogKey(height, rootHash), nil)
	if err != nil {
		return itypes.LayerLog{}, err
	}

	return decodeLayerLog(val)
}

// GetAccountValue gets the persisted account value for given address.
func (s *stateStoreImpl) GetAccountValue(addr common.Address) (itypes.AccountValue, error) {
	res := itypes.AccountValue{}
	val, err := s.ds.Get(getAccountValueKey(addr), nil)
	if err != nil {
		if !errors.Is(err, lerrors.ErrNotFound) {
			return itypes.AccountValue{}, err
		}
		log.Debugf("Get account value empty for %v", addr)
		res.Nonce = 0
		res.Balance = uint256.NewInt(0)
		res.CodeHash = types.EmptyCodeHash
		res.DirtyStorage = false
		// Get account version
		version := uint64(0)
		versionBytes, err := s.ds.Get(getAccountVersionKey(addr), nil)
		if err == nil {
			version = decodeAccountVersion(versionBytes)
		} else if !errors.Is(err, lerrors.ErrNotFound) {
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
	val, err := s.ds.Get(getStorageKey(addr, version, key), nil)
	if err != nil {
		if errors.Is(err, lerrors.ErrNotFound) {
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
	codeBytes, err := s.ds.Get(getCodeKey(codeHash), nil)
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
