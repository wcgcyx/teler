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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/wcgcyx/teler/statestore"
	itypes "github.com/wcgcyx/teler/types"
)

// tempCommittedState is used to track transaction state during processing block.
type tempCommittedState struct {
	sstore  statestore.StateStore
	archive LayeredWorldStateArchive

	// Layered state at last block
	parent    LayeredWorldState
	newHeight uint64

	// Committed states at last transaction
	LogSize            uint
	Logs               map[common.Hash][]*types.Log
	CommittedAccounts  map[common.Address]itypes.AccountValue
	CommittedCodeCount map[common.Hash]int64
	CodePreimage       map[common.Hash][]byte
	CommittedStorage   map[string]map[common.Hash]common.Hash

	// Committed flag
	committed bool
}

// newTempCommittedState creates a new temp committed state.
func newTempCommittedState(
	sstore statestore.StateStore,
	archive LayeredWorldStateArchive,
	parent LayeredWorldState,
	newHeight uint64,
) *tempCommittedState {
	return &tempCommittedState{
		sstore:             sstore,
		archive:            archive,
		parent:             parent,
		newHeight:          newHeight,
		LogSize:            0,
		Logs:               make(map[common.Hash][]*types.Log),
		CommittedAccounts:  make(map[common.Address]itypes.AccountValue),
		CommittedCodeCount: make(map[common.Hash]int64),
		CodePreimage:       make(map[common.Hash][]byte),
		CommittedStorage:   make(map[string]map[common.Hash]common.Hash),
		committed:          false,
	}
}

// GetMutableAccount gets the mutable account of given address.
func (s *tempCommittedState) GetMutableAccount(addr common.Address) (res mutableAccount) {
	log.Debugf("Committed - GetMutableAccount(%v)", addr)
	defer func() {
		log.Debugf("Committed - GetMutableAccount(%v) returns (%v-%v-%v-%v-%v)", addr, res.Nonce(), res.Balance(), res.CodeHash(), res.DirtyStorage(), res.Version())
	}()

	acct, ok := s.CommittedAccounts[addr]
	if !ok {
		acct = s.parent.GetAccountValue(addr, true)
	}
	knownCode := s.CodePreimage[acct.CodeHash]
	knownStorage, ok := s.CommittedStorage[fmt.Sprintf("%v-%v", addr.Hex(), acct.Version)]
	if !ok {
		knownStorage = make(map[common.Hash]common.Hash)
	}
	// Create deep copy of known code and known storage
	var codeCopy []byte
	if knownCode != nil {
		codeCopy = make([]byte, len(knownCode))
		copy(codeCopy, knownCode)
	}
	storageCopy := make(map[common.Hash]common.Hash)
	for k, v := range knownStorage {
		storageCopy[k] = v
	}
	res = newMutableAccountImpl(s.parent, addr, acct, codeCopy, storageCopy)
	return
}

// GetLogs returns the logs matching the specified transaction hash, and annotates
// them with the given blockNumber and blockHash.
func (s *tempCommittedState) GetLogs(hash common.Hash, blockNumber uint64, blockHash common.Hash) (logs []*types.Log) {
	log.Debugf("Committed - GetLogs(%v, %v, %v)", hash, blockNumber, blockHash)
	defer func() {
		str := ""
		for _, l := range logs {
			str += fmt.Sprintf("%v;", l)
		}
		log.Debugf("Committed - GetLogs(%v, %v, %v) returns %v", hash, blockNumber, blockHash, str)
	}()

	var ok bool
	logs, ok = s.Logs[hash]
	if ok {
		for _, l := range logs {
			l.BlockNumber = blockNumber
			l.BlockHash = blockHash
		}
		return
	}
	logs = []*types.Log{}
	return
}

// GetLogSize returns the size of the log.
func (s *tempCommittedState) GetLogSize() (size uint) {
	log.Debugf("Committed - GetLogSize()")
	defer func() { log.Debugf("Committed - GetLogSize() returns %v", size) }()

	size = s.LogSize
	return
}

// ApplyChanges apply the finalised state changes for one transaction.
func (s *tempCommittedState) ApplyChanges(
	txHash common.Hash,
	txIdx int,
	logs []*types.Log,
	dirtyAccounts map[common.Address]mutableAccount,
) {
	log.Debugf("Committed - ApplyChanges(...)")
	defer func() { log.Debugf("Committed - ApplyChanges(...) returns void") }()

	// Apply finalized account changes one by one
	for addr, acct := range dirtyAccounts {
		finalState, knownCode, knownCodeDiff, knownStorage := acct.GetFinalized()
		s.CommittedAccounts[addr] = finalState
		if len(knownCode) > 0 {
			s.CodePreimage[finalState.CodeHash] = knownCode
		}
		for codeHash, diff := range knownCodeDiff {
			s.CommittedCodeCount[codeHash] += diff
		}
		accountKey := itypes.GetAccountStorageKey(addr, finalState.Version)
		for key, val := range knownStorage {
			_, ok := s.CommittedStorage[accountKey]
			if !ok {
				s.CommittedStorage[accountKey] = make(map[common.Hash]common.Hash)
			}
			s.CommittedStorage[accountKey][key] = val
		}
	}

	s.Logs[txHash] = logs
	s.LogSize += uint(len(logs))
}

func (s *tempCommittedState) Commit(block uint64, rootHash common.Hash) (res common.Hash, err error) {
	log.Debugf("Committed - Commit(%v, %v)", block, rootHash)
	defer func() { log.Debugf("Committed - Commit(%v, %v) returns (%v, %v)", block, rootHash, res, err) }()

	res = common.Hash{}
	if s.committed {
		err = fmt.Errorf("already committed")
		return
	}
	if block != s.newHeight {
		err = fmt.Errorf("height mismatch got %v, expect %v", block, s.newHeight)
		return
	}
	// Check if archive has already committed.
	if s.archive.Has(block, rootHash) {
		err = fmt.Errorf("already existed")
		return
	}

	var txn statestore.Transaction
	txn, err = s.sstore.NewTransaction()
	if err != nil {
		return
	}
	defer txn.Discard()

	// Put children
	err = txn.PutChildren(block, rootHash, []common.Hash{})
	if err != nil {
		return
	}
	// Trim code diff
	committedCodeCount := make(map[common.Hash]int64)
	for k, v := range s.CommittedCodeCount {
		if k != types.EmptyCodeHash && v != 0 {
			committedCodeCount[k] = v
		}
	}
	// Put layer log
	err = txn.PutLayerLog(itypes.LayerLog{
		BlockNumber:      block,
		RootHash:         rootHash,
		UpdatedAccounts:  s.CommittedAccounts,
		UpdatedCodeCount: committedCodeCount,
		CodePreimage:     s.CodePreimage,
		UpdatedStorage:   s.CommittedStorage,
	})
	if err != nil {
		return
	}
	err = txn.Commit()
	if err != nil {
		return
	}
	// Add new layered state
	var newLayer LayeredWorldState
	newLayer, err = newLayeredWorldState(s.sstore, s.archive, s.parent, block, rootHash)
	if err != nil {
		return
	}
	s.parent.AddChild(newLayer)
	// Finally try to prune
	newLayer.HandlePrune(make([]common.Hash, 0))

	res = rootHash
	err = nil
	return
}

// Copy creates a deep copy of this state.
func (s *tempCommittedState) Copy() *tempCommittedState {
	logs := make(map[common.Hash][]*types.Log)
	for k, v := range s.Logs {
		temp := make([]*types.Log, len(v))
		copy(temp, v)
		logs[k] = temp
	}
	committedAcct := make(map[common.Address]itypes.AccountValue)
	for k, v := range s.CommittedAccounts {
		committedAcct[k] = itypes.AccountValue{
			Nonce:        v.Nonce,
			Balance:      uint256.NewInt(0).Set(v.Balance),
			CodeHash:     v.CodeHash,
			DirtyStorage: v.DirtyStorage,
			Version:      v.Version,
		}
	}
	committedCodeCount := make(map[common.Hash]int64)
	for k, v := range s.CommittedCodeCount {
		committedCodeCount[k] = v
	}
	codePreimage := make(map[common.Hash][]byte)
	for k, v := range s.CodePreimage {
		codePreimage[k] = append([]byte(nil), v...)
	}
	committedStorage := make(map[string]map[common.Hash]common.Hash)
	for k, v := range s.CommittedStorage {
		committedStorage[k] = make(map[common.Hash]common.Hash)
		for sk, sv := range v {
			committedStorage[k][sk] = sv
		}
	}
	return &tempCommittedState{
		sstore:             s.sstore,
		archive:            s.archive,
		parent:             s.parent,
		newHeight:          s.newHeight,
		LogSize:            s.LogSize,
		Logs:               logs,
		CommittedAccounts:  committedAcct,
		CommittedCodeCount: committedCodeCount,
		CodePreimage:       codePreimage,
		CommittedStorage:   committedStorage,
		committed:          s.committed,
	}
}
