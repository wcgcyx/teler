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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/wcgcyx/teler/statestore"
)

// layeredWorldStateArchiveImpl implements LayeredWorldStateArchive.
type layeredWorldStateArchiveImpl struct {
	opts  Opts
	chain *params.ChainConfig

	lock   sync.RWMutex
	states map[uint64]map[common.Hash]LayeredWorldState
}

// NewLayeredWorldStateArchiveImpl creates a new LayeredWorldStateArchive.
func NewLayeredWorldStateArchiveImpl(
	opts Opts,
	chain *params.ChainConfig,
	sstore statestore.StateStore,
) (LayeredWorldStateArchive, error) {
	res := &layeredWorldStateArchiveImpl{
		chain:  chain,
		opts:   opts,
		lock:   sync.RWMutex{},
		states: make(map[uint64]map[common.Hash]LayeredWorldState),
	}
	_, err := newPersistedWorldState(sstore, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GetChainConfig gets the chain configuration.
func (a *layeredWorldStateArchiveImpl) GetChainConfig() *params.ChainConfig {
	return a.chain
}

// GetMaxLayerToRetain gets the configured max layer to retain in memory.
func (a *layeredWorldStateArchiveImpl) GetMaxLayerToRetain() uint64 {
	return a.opts.MaxLayerToRetain
}

// GetPruningFrequency gets the configured pruning frequency.
func (a *layeredWorldStateArchiveImpl) GetPruningFrequency() uint64 {
	return a.opts.PruningFrequency
}

// Register registers a state in the archive.
func (a *layeredWorldStateArchiveImpl) Register(height uint64, root common.Hash, worldState LayeredWorldState) {
	a.lock.Lock()
	defer a.lock.Unlock()
	hashMap, ok := a.states[height]
	if !ok {
		hashMap = make(map[common.Hash]LayeredWorldState)
		a.states[height] = hashMap
	}
	hashMap[root] = worldState
}

// Deregister deregisters a state in the archive.
func (a *layeredWorldStateArchiveImpl) Deregister(height uint64, root common.Hash) {
	a.lock.Lock()
	defer a.lock.Unlock()
	hashMap, ok := a.states[height]
	if !ok {
		log.Warnf("Attempt to deregister %v-%v where height does not exist", height, root)
		return
	}
	delete(hashMap, root)
	if len(hashMap) == 0 {
		delete(a.states, height)
	}
}

// GetLayared gets the layered world state with given root hash.
func (a *layeredWorldStateArchiveImpl) GetLayared(height uint64, root common.Hash) (LayeredWorldState, error) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	hashMap, ok := a.states[height]
	if !ok {
		return nil, fmt.Errorf("state for %v-%v height does not exist", height, root)
	}
	state, ok := hashMap[root]
	if !ok {
		return nil, fmt.Errorf("state for %v-%v root does not exist", height, root)
	}
	return state, nil
}

// GetMutable gets the mutable world state with given root hash from the layered world state that supports state mutation.
func (a *layeredWorldStateArchiveImpl) GetMutable(height uint64, root common.Hash) (MutableWorldState, error) {
	state, err := a.GetLayared(height, root)
	if err != nil {
		return nil, err
	}
	return state.GetMutable(), nil
}

// Has checks if given state exists.
func (a *layeredWorldStateArchiveImpl) Has(height uint64, root common.Hash) bool {
	a.lock.RLock()
	defer a.lock.RUnlock()
	hashMap, ok := a.states[height]
	if !ok {
		return false
	}
	_, ok = hashMap[root]
	return ok
}
