package worldstate

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
	"github.com/ethereum/go-ethereum/common"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/wcgcyx/teler/statestore"
	itypes "github.com/wcgcyx/teler/types"
)

// layeredStateImpl implements LayeredWorldState.
type layeredStateImpl struct {
	sstore  statestore.StateStore
	archive LayeredWorldStateArchive

	// Layered state at last block
	parent LayeredWorldState

	// Layer log at this block
	layerLog itypes.LayerLog

	children []LayeredWorldState

	// Cache to speed up lookups
	cachedAccts   *lru.Cache[common.Address, itypes.AccountValue]
	cachedCode    *lru.Cache[common.Hash, []byte]
	cachedStorage *lru.Cache[string, common.Hash]
}

// newLayeredWorldState creates a new LayeredWorldState.
func newLayeredWorldState(
	sstore statestore.StateStore,
	archive LayeredWorldStateArchive,
	parent LayeredWorldState,
	height uint64,
	rootHash common.Hash,
) (LayeredWorldState, error) {
	// Get layer log
	layerLog, err := sstore.GetLayerLog(height, rootHash)
	if err != nil {
		return nil, err
	}
	// Create cache
	cachedAccts, err := lru.New[common.Address, itypes.AccountValue](512)
	if err != nil {
		return nil, err
	}
	cachedCode, err := lru.New[common.Hash, []byte](128)
	if err != nil {
		return nil, err
	}
	cachedStorage, err := lru.New[string, common.Hash](4096)
	if err != nil {
		return nil, err
	}
	res := &layeredStateImpl{
		sstore:        sstore,
		archive:       archive,
		parent:        parent,
		layerLog:      layerLog,
		children:      make([]LayeredWorldState, 0),
		cachedAccts:   cachedAccts,
		cachedCode:    cachedCode,
		cachedStorage: cachedStorage,
	}
	// Register state
	archive.Register(height, rootHash, res)
	// Get children
	childrenHashes, err := sstore.GetChildren(height, rootHash)
	if err != nil {
		return nil, err
	}
	for _, childHash := range childrenHashes {
		child, err := newLayeredWorldState(sstore, archive, res, height+1, childHash)
		if err != nil {
			return nil, err
		}
		res.children = append(res.children, child)
	}
	return res, nil
}

// GetAccountValue gets the account value of given address.
func (s *layeredStateImpl) GetAccountValue(addr common.Address, requireCache bool) (res itypes.AccountValue) {
	// log.Debugf("Layer %v - GetAccountValue(%v)", s.layerLog.RootHash, addr)
	// defer func() { log.Debugf("Layer %v - GetAccountValue(%v) returns (%v)", s.layerLog.RootHash, addr, res) }()

	var ok bool
	res, ok = s.layerLog.UpdatedAccounts[addr]
	if !ok {
		res, ok = s.cachedAccts.Get(addr)
		if !ok {
			res = s.parent.GetAccountValue(addr, false)
			if requireCache {
				s.cachedAccts.Add(addr, res)
			}
		}
	}
	return
}

// GetCodeByHash gets the code by code hash.
func (s *layeredStateImpl) GetCodeByHash(codeHash common.Hash, requireCache bool) (code []byte) {
	// log.Debugf("Layer %v - GetCodeByHash(%v)", s.layerLog.RootHash, codeHash)
	// defer func() {
	// 	log.Debugf("Layer %v - GetCodeByHash(%v) returns %v", s.layerLog.RootHash, codeHash, hex.EncodeToString(code))
	// }()

	var ok bool
	code, ok = s.layerLog.CodePreimage[codeHash]
	if !ok {
		code, ok = s.cachedCode.Get(codeHash)
		if !ok {
			code = s.parent.GetCodeByHash(codeHash, false)
			if requireCache {
				s.cachedCode.Add(codeHash, code)
			}
		}
	}
	return
}

// GetStorageByVersion gets storage by addr and version.
func (s *layeredStateImpl) GetStorageByVersion(addr common.Address, version uint64, key common.Hash, requireCache bool) (val common.Hash) {
	// log.Debugf("Layer %v - GetStorageByVersion(%v, %v, %v)", s.layerLog.RootHash, addr, version, key)
	// defer func() {
	// 	log.Debugf("Layer %v - GetStorageByVersion(%v, %v, %v) returns %v", s.layerLog.RootHash, addr, version, key, val)
	// }()

	accountKey := itypes.GetAccountStorageKey(addr, version)
	storage, ok := s.layerLog.UpdatedStorage[accountKey]
	if ok {
		val, ok = storage[key]
		if ok {
			return
		}
	}
	val, ok = s.cachedStorage.Get(accountKey + "-" + key.Hex())
	if !ok {
		val = s.parent.GetStorageByVersion(addr, version, key, false)
		if requireCache {
			s.cachedStorage.Add(accountKey+"-"+key.Hex(), val)
		}
	}
	return
}

// GetReadOnly gets the read only world state from the layered world state.
func (s *layeredStateImpl) GetReadOnly() ReadOnlyWorldState {
	return newMutableWorldState(nil, s.archive, s, s.layerLog.BlockNumber, nil) // TODO: Tracer
}

// GetMutable gets the mutable world state from the layered world state that supports state mutation.
func (s *layeredStateImpl) GetMutable() MutableWorldState {
	return newMutableWorldState(s.sstore, s.archive, s, s.layerLog.BlockNumber+1, nil) // TODO: Tracer
}

// HandlePrune handles the pruning of the state in the state layer.
func (s *layeredStateImpl) HandlePrune(pruningRoute []common.Hash) {
	pruningRoute = append([]common.Hash{s.layerLog.RootHash}, pruningRoute...)
	s.parent.HandlePrune(pruningRoute)
}

// Destruct destructs this layer by removing data associated with this layer from the underlying db.
func (s *layeredStateImpl) Destruct() {
	// Destruct children
	for _, child := range s.children {
		child.Destruct()
	}

	txn, err := s.sstore.NewTransaction()
	if err != nil {
		log.Errorf("Fail to start new transaction to destruct %v-%v: %v", s.layerLog.BlockNumber, s.layerLog.RootHash, err.Error())
		return
	}
	defer txn.Discard()

	// Remove children
	err = txn.DeleteChildren(s.layerLog.BlockNumber, s.layerLog.RootHash)
	if err != nil {
		log.Errorf("Fail to delete children for %v-%v: %v", s.layerLog.BlockNumber, s.layerLog.RootHash, err.Error())
		return
	}

	// Remove layer log
	err = txn.DeleteLayerLog(s.layerLog.BlockNumber, s.layerLog.RootHash)
	if err != nil {
		log.Errorf("Fail to delete layer log for %v-%v: %v", s.layerLog.BlockNumber, s.layerLog.RootHash, err.Error())
		return
	}

	// Commit changes
	err = txn.Commit()
	if err != nil {
		log.Errorf("Fail to commit to destruct %v-%v: %v", s.layerLog.BlockNumber, s.layerLog.RootHash, err.Error())
		return
	}

	// Deregister itself
	s.archive.Deregister(s.layerLog.BlockNumber, s.layerLog.RootHash)
}

// DestructChild destructs given child by removing data associated with this child from the underlying db.
func (s *layeredStateImpl) DestructChild(child common.Hash) {
	// Get child
	index := -1
	for i, childLayer := range s.children {
		if childLayer.GetRootHash().Cmp(child) == 0 {
			index = i
			break
		}
	}
	if index == -1 {
		log.Warnf("Fail to find child state with root hash of %v", child)
		return
	}
	// Destruct child
	s.children[index].Destruct()
	// Update children list
	s.children = append(s.children[:index], s.children[index+1:]...)
	txn, err := s.sstore.NewTransaction()
	if err != nil {
		log.Errorf("Fail to start transaction to remove child %v from %v-%v: %v", child, s.layerLog.BlockNumber, s.layerLog.RootHash, err.Error())
		return
	}
	defer txn.Discard()
	childrenHashes := make([]common.Hash, 0)
	for _, child := range s.children {
		childrenHashes = append(childrenHashes, child.GetRootHash())
	}
	err = txn.PutChildren(s.layerLog.BlockNumber, s.layerLog.RootHash, childrenHashes)
	if err != nil {
		log.Errorf("Fail to write to remove child %v from %v-%v: %v", child, s.layerLog.BlockNumber, s.layerLog.RootHash, err.Error())
		return
	}
	err = txn.Commit()
	if err != nil {
		log.Errorf("Fail to commit to remove child %v from %v-%v: %v", child, s.layerLog.BlockNumber, s.layerLog.RootHash, err.Error())
		return
	}
}

// GetRootHash gets the root hash of this layer.
func (s *layeredStateImpl) GetRootHash() common.Hash {
	return s.layerLog.RootHash
}

// GetHeight gets the current block height of this layer.
func (s *layeredStateImpl) GetHeight() uint64 {
	return s.layerLog.BlockNumber
}

// GetLayerLog gets the log representing this layer.
func (s *layeredStateImpl) GetLayerLog() itypes.LayerLog {
	return s.layerLog
}

// GetChildren gets the list of children.
func (s *layeredStateImpl) GetChildren() []LayeredWorldState {
	return s.children
}

// UpdateParent updates the parent of this layer to given state.
func (s *layeredStateImpl) UpdateParent(parent LayeredWorldState) {
	s.parent = parent
}

// AddChild adds a child to the layer.
func (s *layeredStateImpl) AddChild(child LayeredWorldState) {
	s.children = append(s.children, child)
	txn, err := s.sstore.NewTransaction()
	if err != nil {
		log.Errorf("Fail to start transaction to add child %v to %v-%v: %v", child.GetRootHash(), s.layerLog.BlockNumber, s.layerLog.RootHash, err.Error())
		return
	}
	defer txn.Discard()
	childrenHashes := make([]common.Hash, 0)
	for _, child := range s.children {
		childrenHashes = append(childrenHashes, child.GetRootHash())
	}
	err = txn.PutChildren(s.layerLog.BlockNumber, s.layerLog.RootHash, childrenHashes)
	if err != nil {
		log.Errorf("Fail to write to add child %v to %v-%v: %v", child.GetRootHash(), s.layerLog.BlockNumber, s.layerLog.RootHash, err.Error())
		return
	}
	err = txn.Commit()
	if err != nil {
		log.Errorf("Fail to commit to add child %v to %v-%v: %v", child.GetRootHash(), s.layerLog.BlockNumber, s.layerLog.RootHash, err.Error())
		return
	}
}
