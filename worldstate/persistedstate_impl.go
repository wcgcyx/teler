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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/wcgcyx/teler/statestore"
	itypes "github.com/wcgcyx/teler/types"
)

// persistedStateImpl implements LayeredWorldState that represents the persisted state.
type persistedStateImpl struct {
	sstore  statestore.StateStore
	archive LayeredWorldStateArchive

	// Persisted height and root hash
	height   uint64
	rootHash common.Hash

	children []LayeredWorldState

	// Cache to speed up lookups
	cachedAccts   *lru.Cache[common.Address, itypes.AccountValue]
	cachedCode    *lru.Cache[common.Hash, []byte]
	cachedStorage *lru.Cache[string, common.Hash]
}

// newPersistedWorldState creates a new LayeredWorldState that represents the persisted state.
func newPersistedWorldState(sstore statestore.StateStore, archive LayeredWorldStateArchive) (LayeredWorldState, error) {
	// Get persisted height
	height, rootHash, err := sstore.GetPersistedHeight()
	if err != nil {
		return nil, err
	}
	// Create cache
	cachedAccts, err := lru.New[common.Address, itypes.AccountValue](1024)
	if err != nil {
		return nil, err
	}
	cachedCode, err := lru.New[common.Hash, []byte](256)
	if err != nil {
		return nil, err
	}
	cachedStorage, err := lru.New[string, common.Hash](4096)
	if err != nil {
		return nil, err
	}
	res := &persistedStateImpl{
		sstore:        sstore,
		archive:       archive,
		height:        height,
		rootHash:      rootHash,
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
func (s *persistedStateImpl) GetAccountValue(addr common.Address, _ bool) itypes.AccountValue {
	acct, ok := s.cachedAccts.Get(addr)
	if !ok {
		var err error
		acct, err = s.sstore.GetAccountValue(addr)
		if err != nil {
			log.Errorf("Fail to get account value for %v: %v", addr, err.Error())
			return itypes.AccountValue{}
		}
		s.cachedAccts.Add(addr, acct)
	}
	return acct
}

// GetCodeByHash gets the code by code hash.
func (s *persistedStateImpl) GetCodeByHash(codeHash common.Hash, _ bool) []byte {
	code, ok := s.cachedCode.Get(codeHash)
	if !ok {
		var err error
		code, err = s.sstore.GetCodeByHash(codeHash)
		if err != nil {
			log.Errorf("Fail to get code for %v: %v", codeHash, err.Error())
			return []byte{}
		}
		s.cachedCode.Add(codeHash, code)
	}
	return code
}

// GetStorageByVersion gets storage by addr and version.
func (s *persistedStateImpl) GetStorageByVersion(addr common.Address, version uint64, key common.Hash, _ bool) common.Hash {
	val, ok := s.cachedStorage.Get(itypes.GetAccountStorageKey(addr, version) + "-" + key.Hex())
	if !ok {
		var err error
		val, err = s.sstore.GetStorageByVersion(addr, version, key)
		if err != nil {
			log.Errorf("Fail to get storage for %v-%v-%v: %v", addr, version, key, err.Error())
			return common.Hash{}
		}
		s.cachedStorage.Add(itypes.GetAccountStorageKey(addr, version)+"-"+key.Hex(), val)
	}
	return val
}

// GetReadOnly gets the read only world state from the layered world state.
func (s *persistedStateImpl) GetReadOnly() ReadOnlyWorldState {
	return newMutableWorldState(nil, s.archive, s, s.height, nil) // TODO: Tracer
}

// GetMutable gets the mutable world state from the layered world state that supports state mutation.
func (s *persistedStateImpl) GetMutable() MutableWorldState {
	return newMutableWorldState(s.sstore, s.archive, s, s.height+1, nil) // TODO: Tracer
}

// HandlePrune handles the pruning of the state in the state layer.
func (s *persistedStateImpl) HandlePrune(pruningRoute []common.Hash) {
	if len(s.getToBePruned(pruningRoute)) > 0 {
		oldHeight := s.height
		pruned := 0
		s.cachedAccts.Purge()
		s.cachedCode.Purge()
		s.cachedStorage.Purge()
		for _, toPrune := range s.getToBePruned(pruningRoute) {
			err := s.pruneOne(toPrune)
			if err != nil {
				log.Errorf("Fail to prune %v: %v", toPrune, err.Error())
				return
			}
			pruned++
		}
		log.Infof("Successfully pruned state data from height %v to %v: %v states pruned", oldHeight, s.height, pruned)
	}
}

// getToBePruned gets a list of states to be pruned.
func (s *persistedStateImpl) getToBePruned(pruningRoute []common.Hash) []common.Hash {
	if len(pruningRoute) <= int(s.archive.GetMaxLayerToRetain()) {
		return []common.Hash{}
	}
	return pruningRoute[0:s.archive.GetPruningFrequency()]
}

// pruneOne prunes one child state.
func (s *persistedStateImpl) pruneOne(childHash common.Hash) error {
	var child LayeredWorldState
	// Find child to prune
	for _, c := range s.children {
		if c.GetRootHash().Cmp(childHash) == 0 {
			child = c
			break
		}
	}
	if child == nil {
		log.Warnf("Could not find child %v to prune", childHash)
		return fmt.Errorf("fail to find child %v to prune", childHash)
	}
	// Get layer log
	layerLog := child.GetLayerLog()

	// 1. De-register itself
	s.archive.Deregister(s.height, s.rootHash)

	// 2. Clean old state
	// 2.1. Clean current children
	for _, c := range s.children {
		if c.GetRootHash() != child.GetRootHash() {
			c.Destruct()
		}
	}
	txn, err := s.sstore.NewTransaction()
	if err != nil {
		log.Errorf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard()
	err = txn.DeleteChildren(s.height, s.rootHash)
	if err != nil {
		log.Errorf("Fail to delete children for %v-%v: %v", s.height, s.rootHash, err.Error())
		return err
	}

	// 2.2. Clean Layer log
	err = txn.DeleteLayerLog(s.height+1, child.GetRootHash())
	if err != nil {
		log.Errorf("Fail to delete layer log for %v-%v: %v", s.height, s.rootHash, err.Error())
		return err
	}

	// 2.3. Overwrite persisted state with the new layer log.
	err = txn.PersistLayerLog(layerLog)
	if err != nil {
		log.Errorf("Fail to peresist layer log for %v-%v: %v", s.height, s.rootHash, err.Error())
		return err
	}

	err = txn.Commit()
	if err != nil {
		log.Errorf("Fail to commit transaction to prune %v-%v: %v", s.height, s.rootHash, err.Error())
		return err
	}

	// Update children
	s.children = child.GetChildren()
	for _, child := range s.children {
		child.UpdateParent(s)
	}
	s.rootHash = layerLog.RootHash
	s.height = layerLog.BlockNumber

	// Register itself
	s.archive.Register(s.height, s.rootHash, s)
	return nil
}

// Destruct destructs this layer by removing data associated with this layer from the underlying db.
func (s *persistedStateImpl) Destruct() {
	// This should never happen.
	log.Errorf("attempt to destruct persisted state")
}

// DestructChild destructs given child by removing data associated with this child from the underlying db.
func (s *persistedStateImpl) DestructChild(child common.Hash) {
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
		log.Errorf("Fail to start transaction to remove child %v from %v-%v: %v", child, s.height, s.rootHash, err.Error())
		return
	}
	defer txn.Discard()
	childrenHashes := make([]common.Hash, 0)
	for _, child := range s.children {
		childrenHashes = append(childrenHashes, child.GetRootHash())
	}
	err = txn.PutChildren(s.height, s.rootHash, childrenHashes)
	if err != nil {
		log.Errorf("Fail to write to remove child %v from %v-%v: %v", child, s.height, s.rootHash, err.Error())
		return
	}
	err = txn.Commit()
	if err != nil {
		log.Errorf("Fail to commit to remove child %v from %v-%v: %v", child, s.height, s.rootHash, err.Error())
		return
	}
}

// GetRootHash gets the root hash of this layer.
func (s *persistedStateImpl) GetRootHash() common.Hash {
	// This should never happen.
	log.Errorf("attempt to get root hash of persisted state")
	return s.rootHash
}

// GetHeight gets the current block height of this layer.
func (s *persistedStateImpl) GetHeight() uint64 {
	// This should never happen.
	log.Errorf("attempt to get height of persisted state")
	return s.height
}

// GetLayerLog gets the log representing this layer.
func (s *persistedStateImpl) GetLayerLog() itypes.LayerLog {
	// This should never happen.
	log.Errorf("attempt to get layer log of persisted state")
	return itypes.LayerLog{}
}

// GetChildren gets the list of children.
func (s *persistedStateImpl) GetChildren() []LayeredWorldState {
	// This should never happen.
	log.Errorf("attempt to get children of persisted state")
	return s.children
}

// UpdateParent updates the parent of this layer to given state.
func (s *persistedStateImpl) UpdateParent(parent LayeredWorldState) {
	// This should never happen.
	log.Errorf("attempt to update parent of persisted state")
}

// AddChild adds a child to the layer.
func (s *persistedStateImpl) AddChild(child LayeredWorldState) {
	s.children = append(s.children, child)
	txn, err := s.sstore.NewTransaction()
	if err != nil {
		log.Errorf("Fail to start transaction to add child %v to %v-%v: %v", child.GetRootHash(), s.height, s.rootHash, err.Error())
		return
	}
	defer txn.Discard()
	childrenHashes := make([]common.Hash, 0)
	for _, child := range s.children {
		childrenHashes = append(childrenHashes, child.GetRootHash())
	}
	err = txn.PutChildren(s.height, s.rootHash, childrenHashes)
	if err != nil {
		log.Errorf("Fail to write to add child %v to %v-%v: %v", child.GetRootHash(), s.height, s.rootHash, err.Error())
		return
	}
	err = txn.Commit()
	if err != nil {
		log.Errorf("Fail to commit to add child %v to %v-%v: %v", child.GetRootHash(), s.height, s.rootHash, err.Error())
		return
	}
}
