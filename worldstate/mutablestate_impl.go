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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
	"github.com/wcgcyx/teler/statestore"
)

// mutableStateImpl implements MutableWorldState.
type mutableStateImpl struct {
	newHeight uint64

	// Committed state at last transaction
	parent *tempCommittedState
	config *params.ChainConfig

	// Current state for current transaction
	TxHash         common.Hash
	TxIdx          int
	Refund         uint64
	Logs           []*types.Log
	DirtyAccounts  map[common.Address]mutableAccount
	TransientState map[common.Address]map[common.Hash]common.Hash
	AccessAddress  map[common.Address]int
	AccessStorage  []map[common.Hash]bool
	NewContracts   map[common.Address]bool
	Journals       []*journal

	// Snapshot counter
	snapshotId int

	// Tracer
	logger *tracing.Hooks
}

// journal is used to track the journal of operation.
type journal struct {
	mark   map[int]bool
	revert func()
}

// newMutableWorldState creates a new MutableWorldState.
func newMutableWorldState(
	sstore statestore.StateStore,
	archive LayeredWorldStateArchive,
	parent LayeredWorldState,
	newHeight uint64,
	logger *tracing.Hooks,
) MutableWorldState {
	return &mutableStateImpl{
		newHeight:      newHeight,
		parent:         newTempCommittedState(sstore, archive, parent, newHeight),
		config:         archive.GetChainConfig(),
		TxHash:         common.Hash{},
		TxIdx:          -1,
		Refund:         0,
		Logs:           make([]*types.Log, 0),
		DirtyAccounts:  make(map[common.Address]mutableAccount),
		TransientState: make(map[common.Address]map[common.Hash]common.Hash),
		AccessAddress:  make(map[common.Address]int),
		AccessStorage:  make([]map[common.Hash]bool, 0),
		NewContracts:   make(map[common.Address]bool),
		Journals:       make([]*journal, 0),
		snapshotId:     0,
		logger:         logger,
	}
}

// SetLogger sets the logger for account update hooks.
func (s *mutableStateImpl) SetLogger(l *tracing.Hooks) {
	s.logger = l
}

// loadAccountValue for given address from dirty states or parent.
func (s *mutableStateImpl) loadAccount(addr common.Address, write bool) (acct mutableAccount) {
	// log.Debugf("Mutable - loadAccount(%v, %v)", addr, write)
	// defer func() {
	// 	log.Debugf("Mutable - loadAccount(%v, %v) returns (%v-%v-%v-%v-%v)", addr, write, acct.Nonce(), acct.Balance(), acct.CodeHash(), acct.DirtyStorage(), acct.Version())
	// }()

	// Check dirty first
	var ok bool
	acct, ok = s.DirtyAccounts[addr]
	if !ok {
		// log.Debugf("Mutable - loadAccount(%v, %v), load from parent", addr, write)
		acct = s.parent.GetMutableAccount(addr)
		if write {
			// log.Debugf("Mutable - loadAccount(%v, %v), load to dirty accounts", addr, write)
			s.DirtyAccounts[addr] = acct
			s.recordJournal(func() { delete(s.DirtyAccounts, addr) })
		}
	}
	return
}

// recordJournal is used to record a revert function.
func (s *mutableStateImpl) recordJournal(revert func()) {
	// Add revert to journal
	s.Journals = append(s.Journals, &journal{
		mark:   make(map[int]bool),
		revert: revert,
	})
}

// Exist reports whether the given account address exists in the state.
// Notably this also returns true for self-destructed accounts.
func (s *mutableStateImpl) Exist(addr common.Address) (exists bool) {
	// log.Debugf("Mutable - Exist(%v)", addr)
	// defer func() { log.Debugf("Mutable - Exist(%v) returns %v", addr, exists) }()

	acct := s.loadAccount(addr, false)
	exists = acct.Exists(s.config.IsEIP158(big.NewInt(int64(s.newHeight))))
	return
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0).
func (s *mutableStateImpl) Empty(addr common.Address) (empty bool) {
	// log.Debugf("Mutable - Empty(%v)", addr)
	// defer func() { log.Debugf("Mutable - Empty(%v) returns %v", addr, empty) }()

	acct := s.loadAccount(addr, false)
	empty = acct.Empty()
	return
}

// GetBalance retrieves the balance from the given address or 0 if object not found.
func (s *mutableStateImpl) GetBalance(addr common.Address) (balance *uint256.Int) {
	// log.Debugf("Mutable - GetBalance(%v)", addr)
	// defer func() { log.Debugf("Mutable - GetBalance(%v) returns %v", addr, balance) }()

	acct := s.loadAccount(addr, false)
	balance = uint256.NewInt(0).Set(acct.Balance())
	return
}

// GetNonce retrieves the nonce from the given address or 0 if object not found.
func (s *mutableStateImpl) GetNonce(addr common.Address) (nonce uint64) {
	// log.Debugf("Mutable - GetNonce(%v)", addr)
	// defer func() { log.Debugf("Mutable - GetNonce(%v) returns %v", addr, nonce) }()

	acct := s.loadAccount(addr, false)
	nonce = acct.Nonce()
	return
}

// GetCodeHash gets the code hash of the given address.
func (s *mutableStateImpl) GetCodeHash(addr common.Address) (codeHash common.Hash) {
	// log.Debugf("Mutable - GetCodeHash(%v)", addr)
	// defer func() { log.Debugf("Mutable - GetCodeHash(%v) returns %v", addr, codeHash) }()

	acct := s.loadAccount(addr, false)
	codeHash = acct.CodeHash()
	return
}

// GetCode gets the code of the given address.
func (s *mutableStateImpl) GetCode(addr common.Address) (code []byte) {
	// log.Debugf("Mutable - GetCode(%v)", addr)
	// defer func() { log.Debugf("Mutable - GetCode(%v) returns %v", addr, hex.EncodeToString(code)) }()

	acct := s.loadAccount(addr, false)
	code = acct.Code()
	return
}

// GetCodeSize gets the size of the code of the given address.
func (s *mutableStateImpl) GetCodeSize(addr common.Address) (size int) {
	// log.Debugf("Mutable - GetCodeSize(%v)", addr)
	// defer func() { log.Debugf("Mutable - GetCodeSize(%v) returns %v", addr, size) }()

	size = len(s.GetCode(addr))
	return
}

// GetStorageRoot retrieves the storage root from the given address or empty
// if object not found.
func (s *mutableStateImpl) GetStorageRoot(addr common.Address) (storageRoot common.Hash) {
	// log.Debugf("Mutable - GetStorageRoot(%v)", addr)
	// defer func() { log.Debugf("Mutable - GetStorageRoot(%v) returns %v", addr, storageRoot) }()

	acct := s.loadAccount(addr, false)
	storageRoot = types.EmptyRootHash
	if acct.DirtyStorage() {
		// Note: Return non-empty storage root, as it is only used by EIP-7610
		storageRoot = common.HexToHash("1")
	}
	return
}

// GetState retrieves the value associated with the specific key.
func (s *mutableStateImpl) GetState(addr common.Address, key common.Hash) (val common.Hash) {
	// log.Debugf("Mutable - GetState(%v, %v)", addr, key)
	// defer func() { log.Debugf("Mutable - GetState(%v, %v) returns %v", addr, key, val) }()

	acct := s.loadAccount(addr, false)
	val = acct.GetState(key)
	return
}

// SetTxContext sets the current transaction hash and index which are
// used when the EVM emits new state logs. It should be invoked before
// transaction execution.
func (s *mutableStateImpl) SetTxContext(thash common.Hash, ti int) {
	// log.Debugf("Mutable - SetTxContext(%v, %v)", thash, ti)
	// defer func() { log.Debugf("Mutable - SetTxContext(%v, %v) returns void", thash, ti) }()

	// Note: This does not involve a journal revert in the original geth source code
	s.TxHash = thash
	s.TxIdx = ti
	s.Refund = 0
	s.Logs = make([]*types.Log, 0)
}

// TxIndex returns the current transaction index set by Prepare.
func (s *mutableStateImpl) TxIndex() (txID int) {
	// log.Debugf("Mutable - TxIndex()")
	// defer func() { log.Debugf("Mutable - TxIndex() returns %v", txID) }()

	txID = s.TxIdx
	return
}

// Prepare handles the preparatory steps for executing a state transition with.
// This method must be invoked before state transition.
//
// Berlin fork:
// - Add sender to access list (2929)
// - Add destination to access list (2929)
// - Add precompiles to access list (2929)
// - Add the contents of the optional tx access list (2930)
//
// Potential EIPs:
// - Reset access list (Berlin)
// - Add coinbase to access list (EIP-3651)
// - Reset transient storage (EIP-1153)
func (s *mutableStateImpl) Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	// log.Debugf("Mutable - Prepare(...)")
	// log.Debugf("ChainID:%v Homestead:%v EIP150:%v EIP155:%v EIP158:%v", rules.ChainID, rules.IsHomestead, rules.IsEIP150, rules.IsEIP155, rules.IsEIP158)
	// log.Debugf("EIP2929:%v EIP4762:%v", rules.IsEIP2929, rules.IsEIP4762)
	// log.Debugf("Byzantium:%v Constantinople:%v Petersburg:%v Istanbul:%v", rules.IsByzantium, rules.IsConstantinople, rules.IsPetersburg, rules.IsIstanbul)
	// log.Debugf("Berlin:%v London:%v", rules.IsBerlin, rules.IsLondon)
	// log.Debugf("Merge:%v Shanghai:%v Cancun:%v Prague:%v Verkle:%v", rules.IsMerge, rules.IsShanghai, rules.IsCancun, rules.IsPrague, rules.IsVerkle)
	// log.Debugf("sender:%v coinbase: %v dest: %v", sender, coinbase, dest)
	// log.Debugf("precompiles: %v", precompiles)
	// log.Debugf("txAccesses: %v", txAccesses)

	// Adapted from go-ethereum@v1.14.8/core/state/statedb.go
	if rules.IsEIP2929 && rules.IsEIP4762 {
		log.Panicf("eip2929 and eip4762 are both activated")
	}
	if rules.IsEIP2929 {
		// Clear out any leftover from previous executions
		s.AccessAddress = make(map[common.Address]int)
		s.AccessStorage = make([]map[common.Hash]bool, 0)

		s.AddAddressToAccessList(sender)
		if dest != nil {
			s.AddAddressToAccessList(*dest)
			// If it's a create-tx, the destination will be added inside evm.create
		}
		for _, addr := range precompiles {
			s.AddAddressToAccessList(addr)
		}
		for _, el := range txAccesses {
			s.AddAddressToAccessList(el.Address)
			for _, key := range el.StorageKeys {
				s.AddSlotToAccessList(el.Address, key)
			}
		}
		if rules.IsShanghai { // EIP-3651: warm coinbase
			s.AddAddressToAccessList(coinbase)
		}
	}
	// Reset transient storage at the beginning of transaction execution
	s.TransientState = make(map[common.Address]map[common.Hash]common.Hash)
}

// CreateAccount explicitly creates a new state object, assuming that the
// account did not previously exist in the state. If the account already
// exists, this function will silently overwrite it which might lead to a
// consensus bug eventually.
func (s *mutableStateImpl) CreateAccount(addr common.Address) {
	// log.Debugf("Mutable - CreateAccount(%v)", addr)
	// defer func() { log.Debugf("Mutable - CreateAccount(%v) returns void", addr) }()

	acct := s.loadAccount(addr, true)
	s.recordJournal(acct.Create())
}

// CreateContract is used whenever a contract is created. This may be preceded
// by CreateAccount, but that is not required if it already existed in the
// state due to funds sent beforehand.
// This operation sets the 'newContract'-flag, which is required in order to
// correctly handle EIP-6780 'delete-in-same-transaction' logic.
func (s *mutableStateImpl) CreateContract(addr common.Address) {
	// log.Debugf("Mutable - CreateContract(%v)", addr)
	// defer func() { log.Debugf("Mutable - CreateContract(%v) returns void", addr) }()

	_, ok := s.NewContracts[addr]
	if !ok {
		s.NewContracts[addr] = true
		s.recordJournal(func() { delete(s.NewContracts, addr) })
	}
}

// SubBalance subtracts amount from the account associated with addr.
func (s *mutableStateImpl) SubBalance(addr common.Address, amt *uint256.Int, reason tracing.BalanceChangeReason) {
	// log.Debugf("Mutable - SubBalance(%v, %v)", addr, amt)
	// defer func() { log.Debugf("Mutable - SubBalance(%v, %v) returns void", addr, amt) }()

	acct := s.loadAccount(addr, true)
	original := uint256.NewInt(0).Set(acct.Balance())
	new := uint256.NewInt(0).Sub(original, amt)
	if new.Sign() < 0 {
		new.Set(uint256.NewInt(0))
	}
	s.recordJournal(acct.SetBalance(new))

	if s.logger != nil && s.logger.OnBalanceChange != nil {
		s.logger.OnBalanceChange(addr, original.ToBig(), new.ToBig(), reason)
	}
}

// AddBalance adds amount to the account associated with addr.
func (s *mutableStateImpl) AddBalance(addr common.Address, amt *uint256.Int, reason tracing.BalanceChangeReason) {
	// log.Debugf("Mutable - AddBalance(%v, %v)", addr, amt)
	// defer func() { log.Debugf("Mutable - AddBalance(%v, %v) returns void", addr, amt) }()

	acct := s.loadAccount(addr, true)
	original := uint256.NewInt(0).Set(acct.Balance())
	new := uint256.NewInt(0).Add(original, amt)
	s.recordJournal(acct.SetBalance(new))

	if s.logger != nil && s.logger.OnBalanceChange != nil {
		s.logger.OnBalanceChange(addr, original.ToBig(), new.ToBig(), reason)
	}
}

// SetBalance sets the balance of the account associated with addr.
func (s *mutableStateImpl) SetBalance(addr common.Address, amt *uint256.Int, reason tracing.BalanceChangeReason) {
	// log.Debugf("Mutable - SetBalance(%v, %v)", addr, amt)
	// defer func() { log.Debugf("Mutable - SetBalance(%v, %v) returns void", addr, amt) }()

	acct := s.loadAccount(addr, true)
	original := uint256.NewInt(0).Set(acct.Balance())
	new := amt
	s.recordJournal(acct.SetBalance(new))

	if s.logger != nil && s.logger.OnBalanceChange != nil {
		s.logger.OnBalanceChange(addr, original.ToBig(), new.ToBig(), reason)
	}
}

// SetNonce sets the given nonce to the given address.
func (s *mutableStateImpl) SetNonce(addr common.Address, nonce uint64) {
	// log.Debugf("Mutable - SetNonce(%v, %v)", addr, nonce)
	// defer func() { log.Debugf("Mutable - SetNonce(%v, %v) returns void", addr, nonce) }()

	acct := s.loadAccount(addr, true)

	if s.logger != nil && s.logger.OnNonceChange != nil {
		s.logger.OnNonceChange(addr, acct.Nonce(), nonce)
	}

	s.recordJournal(acct.SetNonce(nonce))
}

// SetCode sets the code to the given address.
func (s *mutableStateImpl) SetCode(addr common.Address, code []byte) {
	// log.Debugf("Mutable - SetCode(%v, %v)", addr, code)
	// defer func() { log.Debugf("Mutable - SetCode(%v, %v) returns void", addr, code) }()

	acct := s.loadAccount(addr, true)

	if s.logger != nil && s.logger.OnCodeChange != nil {
		prevcode := acct.Code()
		s.logger.OnCodeChange(addr, acct.CodeHash(), prevcode, crypto.Keccak256Hash(code), code)
	}

	s.recordJournal(acct.SetCode(code))
}

// SetState sets the value associated with the specific key.
func (s *mutableStateImpl) SetState(addr common.Address, key common.Hash, val common.Hash) {
	// log.Debugf("Mutable - SetState(%v, %v, %v)", addr, key, val)
	// defer func() { log.Debugf("Mutable - SetState(%v, %v, %v) returns void", addr, key, val) }()

	acct := s.loadAccount(addr, true)

	if s.logger != nil && s.logger.OnStorageChange != nil {
		prev := acct.GetState(key)
		s.logger.OnStorageChange(addr, key, prev, val)
	}

	s.recordJournal(acct.SetState(key, val))
}

// GetCommittedState retrieves the value associated with the specific key
// without any mutations caused in the current execution.
func (s *mutableStateImpl) GetCommittedState(addr common.Address, key common.Hash) (val common.Hash) {
	// log.Debugf("Mutable - GetCommittedState(%v, %v)", addr, key)
	// defer func() { log.Debugf("Mutable - GetCommittedState(%v, %v) returns %v", addr, key, val) }()

	val = s.parent.GetMutableAccount(addr).GetState(key)
	return
}

// GetTransientState gets transient storage for a given account.
func (s *mutableStateImpl) GetTransientState(addr common.Address, key common.Hash) (val common.Hash) {
	// log.Debugf("Mutable - GetTransientState(%v, %v)", addr, key)
	// defer func() { log.Debugf("Mutable - GetTransientState(%v, %v) returns %v", addr, key, val) }()

	storageMap, ok := s.TransientState[addr]
	if ok {
		val, ok = storageMap[key]
		if ok {
			return
		}
	}
	val = common.Hash{}
	return
}

// SetTransientState sets transient storage for a given account. It
// adds the change to the journal so that it can be rolled back
// to its previous value if there is a revert.
func (s *mutableStateImpl) SetTransientState(addr common.Address, key common.Hash, val common.Hash) {
	// log.Debugf("Mutable - SetTransientState(%v, %v, %v)", addr, key, val)
	// defer func() { log.Debugf("Mutable - SetTransientState(%v, %v, %v) returns void", addr, key, val) }()

	storageMap, ok := s.TransientState[addr]
	if !ok {
		storageMap = make(map[common.Hash]common.Hash)
		s.TransientState[addr] = storageMap
		s.recordJournal(func() { delete(s.TransientState, addr) })
	}
	original, ok := storageMap[key]
	if ok {
		storageMap[key] = val
		s.recordJournal(func() { storageMap[key] = original })
		return
	}
	storageMap[key] = val
	s.recordJournal(func() { delete(storageMap, key) })
}

// GetRefund returns the current value of the refund counter.
func (s *mutableStateImpl) GetRefund() (refund uint64) {
	// log.Debugf("Mutable - GetRefund()")
	// defer func() { log.Debugf("Mutable - GetRefund() returns %v", refund) }()

	refund = s.Refund
	return
}

// AddRefund adds gas to the refund counter.
func (s *mutableStateImpl) AddRefund(gas uint64) {
	// log.Debugf("Mutable - AddRefund(%v)", gas)
	// defer func() { log.Debugf("Mutable - AddRefund(%v) returns void", gas) }()

	original := s.Refund
	s.Refund += gas
	s.recordJournal(func() { s.Refund = original })
}

// SubRefund removes gas from the refund counter.
// This method will panic if the refund counter goes below zero.
func (s *mutableStateImpl) SubRefund(gas uint64) {
	// log.Debugf("Mutable - SubRefund(%v)", gas)
	// defer func() { log.Debugf("Mutable - SubRefund(%v) returns void", gas) }()

	original := s.Refund
	if s.Refund < gas {
		log.Panicf("refund counter goes below zero")
	}
	s.Refund -= gas
	s.recordJournal(func() { s.Refund = original })
}

// Check if given account was marked as self-destructed.
func (s *mutableStateImpl) HasSelfDestructed(addr common.Address) (destructed bool) {
	// log.Debugf("Mutable - HasSelfDestructed(%v)", addr)
	// defer func() { log.Debugf("Mutable - HasSelfDestructed(%v) returns %v", addr, destructed) }()

	// Note: even though this is a read only query,
	// but this account should pre exist in dirty accounts.
	acct := s.loadAccount(addr, false)
	destructed = acct.HasSelfDestructed()
	return
}

// SelfDestruct marks the given account as selfdestructed.
// This clears the account balance.
//
// The account's state object is still available until the state is committed,
// getStateObject will return a non-nil account after SelfDestruct.
func (s *mutableStateImpl) SelfDestruct(addr common.Address) {
	// log.Debugf("Mutable - SelfDestruct(%v)", addr)
	// defer func() { log.Debugf("Mutable - SelfDestruct(%v) returns void", addr) }()

	acct := s.loadAccount(addr, true)

	if s.logger != nil && s.logger.OnBalanceChange != nil && acct.Balance().Sign() > 0 {
		s.logger.OnBalanceChange(addr, acct.Balance().ToBig(), big.NewInt(0), tracing.BalanceDecreaseSelfdestruct)
	}

	s.recordJournal(acct.SelfDestruct())
}

// SelfDestruct given account according to EIP-6780.
func (s *mutableStateImpl) Selfdestruct6780(addr common.Address) {
	// log.Debugf("Mutable - Selfdestruct6780(%v)", addr)
	// defer func() { log.Debugf("Mutable - Selfdestruct6780(%v) returns void", addr) }()

	acct := s.loadAccount(addr, true)
	_, ok := s.NewContracts[addr]
	if ok {
		acct.SelfDestruct()
	}
}

// AddressInAccessList returns true if the given address is in the access list.
func (s *mutableStateImpl) AddressInAccessList(addr common.Address) (ok bool) {
	// log.Debugf("Mutable - AddressInAccessList(%v)", addr)
	// defer func() { log.Debugf("Mutable - AddressInAccessList(%v) returns %v", addr, ok) }()

	_, ok = s.AccessAddress[addr]
	return
}

// AddAddressToAccessList adds the given address to the access list.
func (s *mutableStateImpl) AddAddressToAccessList(addr common.Address) {
	// log.Debugf("Mutable - AddAddressToAccessList(%v)", addr)
	// defer func() { log.Debugf("Mutable - AddAddressToAccessList(%v) returns void", addr) }()

	_, ok := s.AccessAddress[addr]
	if ok {
		return
	}
	s.AccessAddress[addr] = -1
	s.recordJournal(func() { delete(s.AccessAddress, addr) })
}

// SlotInAccessList returns true if the given (address, slot)-tuple is in the access list.
func (s *mutableStateImpl) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	// log.Debugf("Mutable - SlotInAccessList(%v, %v)", addr, slot)
	// defer func() { log.Debugf("Mutable - SlotInAccessList(%v, %v) returns %v, %v", addr, slot, addressOk, slotOk) }()

	idx, ok := s.AccessAddress[addr]
	if !ok {
		addressOk = false
		slotOk = false
		return
	}
	if idx == -1 {
		addressOk = true
		slotOk = false
		return
	}
	_, ok = s.AccessStorage[idx][slot]
	addressOk = true
	slotOk = ok
	return
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list.
func (s *mutableStateImpl) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	// log.Debugf("Mutable - AddSlotToAccessList(%v, %v)", addr, slot)
	// defer func() { log.Debugf("Mutable - AddSlotToAccessList(%v, %v) returns void", addr, slot) }()

	idx, ok := s.AccessAddress[addr]
	if !ok {
		s.AccessAddress[addr] = len(s.AccessStorage)
		s.AccessStorage = append(s.AccessStorage, make(map[common.Hash]bool))
		s.recordJournal(func() {
			s.AccessStorage = s.AccessStorage[:len(s.AccessStorage)-1]
			delete(s.AccessAddress, addr)
		})
	} else if idx == -1 {
		s.AccessAddress[addr] = len(s.AccessStorage)
		s.AccessStorage = append(s.AccessStorage, make(map[common.Hash]bool))
		s.recordJournal(func() {
			s.AccessStorage = s.AccessStorage[:len(s.AccessStorage)-1]
			s.AccessAddress[addr] = -1
		})
	}
	idx = s.AccessAddress[addr]
	_, ok = s.AccessStorage[idx][slot]
	if !ok {
		s.AccessStorage[idx][slot] = true
		s.recordJournal(func() { delete(s.AccessStorage[idx], slot) })
	}
}

// PointCache returns the point cache used in computations.
func (s *mutableStateImpl) PointCache() *utils.PointCache {
	// TODO: This will become problematic after EIP-4762, Need to fix.
	log.Panicf("Not implemented.")
	return nil
}

// GetLogs returns the logs matching the specified transaction hash, and annotates
// them with the given blockNumber and blockHash.
func (s *mutableStateImpl) GetLogs(hash common.Hash, blockNumber uint64, blockHash common.Hash) (logs []*types.Log) {
	// log.Debugf("Mutable - GetLogs(%v, %v, %v)", hash, blockNumber, blockHash)
	defer func() {
		str := ""
		for _, l := range logs {
			str += fmt.Sprintf("%v;", l)
		}
		// log.Debugf("Mutable - GetLogs(%v, %v, %v) returns %v", hash, blockNumber, blockHash, str)
	}()

	if s.TxHash.Cmp(hash) == 0 {
		for _, l := range s.Logs {
			// Note: This does not involve a journal revert in the original geth source code
			l.BlockNumber = blockNumber
			l.BlockHash = blockHash
		}
		logs = s.Logs
		return
	}
	logs = s.parent.GetLogs(hash, blockNumber, blockHash)
	return
}

// Add log adds a log to the log list.
func (s *mutableStateImpl) AddLog(logs *types.Log) {
	// log.Debugf("Mutable - AddLog(%v)", logs)
	// defer func() { log.Debugf("Mutable - AddLog(%v) returns void", logs) }()

	logs.TxHash = s.TxHash
	logs.TxIndex = uint(s.TxIdx)
	logs.Index = s.parent.GetLogSize() + uint(len(s.Logs))
	s.Logs = append(s.Logs, logs)
	s.recordJournal(func() { s.Logs = s.Logs[:len(s.Logs)-1] })

	if s.logger != nil && s.logger.OnLog != nil {
		s.logger.OnLog(logs)
	}
}

// AddPreimage records a SHA3 preimage seen by the VM.
func (s *mutableStateImpl) AddPreimage(key common.Hash, val []byte) {
	// log.Debugf("Mutable - AddPreimage(%v, %v)", key, hex.EncodeToString(val))
	// defer func() { log.Debugf("Mutable - AddPreimage(%v, %v) returns void", key, hex.EncodeToString(val)) }()
	// Not supported.
}

// Witness retrieves the current state witness being collected.
func (s *mutableStateImpl) Witness() *stateless.Witness {
	// log.Debugf("Mutable - Witness()")
	// defer func() { log.Debugf("Mutable - Witness() returns nil") }()

	// Not supported.
	return nil
}

// Snapshot returns an identifier for the current revision of the state.
func (s *mutableStateImpl) Snapshot() (res int) {
	// log.Debugf("Mutable - Snapshot()")
	// defer func() { log.Debugf("Mutable - Snapshot() returns %v", res) }()

	res = s.snapshotId
	s.snapshotId++
	if len(s.Journals) == 0 {
		s.Journals = append(s.Journals, &journal{
			mark:   make(map[int]bool),
			revert: func() {},
		})
	}
	s.Journals[len(s.Journals)-1].mark[res] = true
	return
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (s *mutableStateImpl) RevertToSnapshot(ti int) {
	// log.Debugf("Mutable - RevertToSnapshot(%v)", ti)
	// defer func() { log.Debugf("Mutable - RevertToSnapshot(%v) returns void", ti) }()

	for i := len(s.Journals) - 1; i >= 0; i-- {
		entry := s.Journals[i]
		if entry.mark[ti] {
			// 1. Delete this snapshot.
			delete(entry.mark, ti)
			// 2. Apply reverts and update journals
			reverts := s.Journals[i+1:]
			s.Journals = s.Journals[:i+1]
			for j := len(reverts) - 1; j >= 0; j-- {
				reverts[j].revert()
			}
			// 3. Return
			return
		}
	}
	log.Panicf("cannot revert to snapshot")
}

// Finalise finalises the state by removing the destructed objects and clears
// the journal as well as the refunds. Finalise, however, will not push any updates
// into the tries just yet. Only IntermediateRoot or Commit will do that.
func (s *mutableStateImpl) Finalise(deleteEmptyObjects bool) {
	// log.Debugf("Mutable - Finalise(%v)", deleteEmptyObjects)
	// defer func() { log.Debugf("Mutable - Finalise(%v) returns void", deleteEmptyObjects) }()

	for addr, acct := range s.DirtyAccounts {
		if deleteEmptyObjects && acct.Empty() {
			acct.SelfDestruct()
		}

		// If ether was sent to account post-selfdestruct it is burnt.
		if bal := acct.Balance(); s.logger != nil && s.logger.OnBalanceChange != nil && acct.HasSelfDestructed() && bal.Sign() != 0 {
			s.logger.OnBalanceChange(addr, bal.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestructBurn)
		}
	}

	s.parent.ApplyChanges(s.TxHash, s.TxIdx, s.Logs, s.DirtyAccounts)

	s.DirtyAccounts = make(map[common.Address]mutableAccount)
	s.TransientState = make(map[common.Address]map[common.Hash]common.Hash)
	s.AccessAddress = make(map[common.Address]int)
	s.AccessStorage = make([]map[common.Hash]bool, 0)
	s.NewContracts = make(map[common.Address]bool)
	s.Journals = make([]*journal, 0)
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (s *mutableStateImpl) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	// log.Debugf("Mutable - IntermediateRoot(%v)", deleteEmptyObjects)
	// defer func() {
	// 	log.Debugf("Mutable - IntermediateRoot(%v) returns %v", deleteEmptyObjects, types.EmptyRootHash)
	// }()

	s.Finalise(deleteEmptyObjects)
	// Not supported, so empty root hash is returned
	return types.EmptyRootHash
}

// Commit writes the state mutations into the configured data stores.
//
// Once the state is committed, tries cached in stateDB (including account
// trie, storage tries) will no longer be functional. A new state instance
// must be created with new root and updated database for accessing post-
// commit states.
//
// The associated block number of the state transition is also provided
// for more chain context.
func (s *mutableStateImpl) Commit(block uint64, rootHash common.Hash, deleteEmptyObjects bool) (res common.Hash, err error) {
	// log.Debugf("Mutable - Commit(%v, %v, %v)", block, rootHash, deleteEmptyObjects)
	// defer func() {
	// 	log.Debugf("Mutable - Commit(%v, %v, %v) returns (%v, %v)", block, rootHash, deleteEmptyObjects, res, err)
	// }()

	s.IntermediateRoot(deleteEmptyObjects)
	res, err = s.parent.Commit(block, rootHash)
	return
}

// Copy creates a deep copy of this state.
func (s *mutableStateImpl) Copy() MutableWorldState {
	logs := make([]*types.Log, len(s.Logs))
	copy(logs, s.Logs)
	dirtyAccts := make(map[common.Address]mutableAccount)
	for k, v := range s.DirtyAccounts {
		dirtyAccts[k] = v.Copy()
	}
	transientState := make(map[common.Address]map[common.Hash]common.Hash)
	for k, v := range s.TransientState {
		transientState[k] = make(map[common.Hash]common.Hash)
		for sk, sv := range v {
			transientState[k][sk] = sv
		}
	}
	accessAddress := make(map[common.Address]int)
	for k, v := range s.AccessAddress {
		accessAddress[k] = v
	}
	accessStorage := make([]map[common.Hash]bool, 0)
	for k, v := range s.AccessStorage {
		accessStorage[k] = make(map[common.Hash]bool)
		for sk, sv := range v {
			accessStorage[k][sk] = sv
		}
	}
	newContracts := make(map[common.Address]bool)
	for k, v := range s.NewContracts {
		newContracts[k] = v
	}
	journals := make([]*journal, len(s.Journals))
	copy(journals, s.Journals)
	return &mutableStateImpl{
		newHeight:      s.newHeight,
		parent:         s.parent.Copy(),
		config:         s.config,
		TxHash:         s.TxHash,
		TxIdx:          s.TxIdx,
		Refund:         s.Refund,
		Logs:           logs,
		DirtyAccounts:  dirtyAccts,
		TransientState: transientState,
		AccessAddress:  accessAddress,
		AccessStorage:  accessStorage,
		NewContracts:   newContracts,
		Journals:       journals,
		snapshotId:     s.snapshotId,
		logger:         s.logger,
	}
}
