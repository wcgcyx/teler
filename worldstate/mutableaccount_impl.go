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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	itypes "github.com/wcgcyx/teler/types"
)

// Constants to represent the state of the account
// There are three possible states:
// 1. Account does not exist, which is when version % 3 == 0
// 2. Account is being updated, which is when version % 3 == 1
// 3. Account is being deleted, wihch is when version % 3 == 2
const (
	notExisted   = uint64(0)
	updating     = uint64(1)
	beingDeleted = uint64(2)
	versions     = uint64(3)
)

// mutableAccountImpl implements Account interface
type mutableAccountImpl struct {
	// parent state
	parent LayeredWorldState

	// account value
	addr         common.Address
	nonce        uint64
	balance      *uint256.Int
	codeHash     common.Hash
	dirtyStorage bool
	version      uint64

	// code
	knownCode     []byte // lazy-init
	knownCodeDiff map[common.Hash]int64

	// storage
	knownStorage map[common.Hash]common.Hash // lazy-init
}

func newMutableAccountImpl(
	parent LayeredWorldState,
	addr common.Address,
	acct itypes.AccountValue,
	knownCode []byte,
	knownStorage map[common.Hash]common.Hash,
) mutableAccount {
	return &mutableAccountImpl{
		parent:        parent,
		addr:          addr,
		nonce:         acct.Nonce,
		balance:       uint256.NewInt(0).Set(acct.Balance),
		codeHash:      acct.CodeHash,
		dirtyStorage:  acct.DirtyStorage,
		version:       acct.Version,
		knownCode:     knownCode,
		knownCodeDiff: make(map[common.Hash]int64),
		knownStorage:  knownStorage,
	}
}

// interpretVersion is used to interpret the version of this account to be either:
// 1. not existed, or 2. updating, or 3. being deleted.
func (acct *mutableAccountImpl) interpretVersion() uint64 {
	return acct.version % versions
}

// Empty returns if this account is empty.
func (acct *mutableAccountImpl) Empty() (empty bool) {
	log.Debugf("Account %v - Empty()", acct.addr)
	defer func() { log.Debugf("Account %v - Empty() returns %v", acct.addr, empty) }()

	empty = acct.nonce == 0 &&
		acct.balance.IsZero() &&
		acct.codeHash.Cmp(types.EmptyCodeHash) == 0
	return
}

// Exist returns if this account exists.
func (acct *mutableAccountImpl) Exists(isEIP158 bool) (exists bool) {
	log.Debugf("Account %v - Exists()", acct.addr)
	defer func() { log.Debugf("Account %v - Exists() returns %v", acct.addr, exists) }()

	if acct.interpretVersion() == notExisted {
		exists = false
		return
	}
	if isEIP158 {
		exists = !acct.Empty()
		return
	}
	exists = true
	return
}

// Nonce returns the nonce of this account.
func (acct *mutableAccountImpl) Nonce() uint64 {
	return acct.nonce
}

// Balance returns the balance of this account.
func (acct *mutableAccountImpl) Balance() *uint256.Int {
	return acct.balance
}

// CodeHash returns the code hash of this account.
func (acct *mutableAccountImpl) CodeHash() common.Hash {
	return acct.codeHash
}

// Code returns the code of this account.
func (acct *mutableAccountImpl) Code() []byte {
	if acct.codeHash.Cmp(types.EmptyCodeHash) == 0 {
		return []byte{}
	}
	if acct.knownCode == nil {
		acct.knownCode = acct.parent.GetCodeByHash(acct.codeHash, true)
	}
	return acct.knownCode
}

// DirtyStorage returns if account's storage trie is empty.
func (acct *mutableAccountImpl) DirtyStorage() bool {
	return acct.dirtyStorage
}

// GetState returns the given storage slot value of this account.
func (acct *mutableAccountImpl) GetState(key common.Hash) (res common.Hash) {
	log.Debugf("Account %v - GetState(%v)", acct.addr, key)
	defer func() { log.Debugf("Account %v - GetState(%v) returns %v", acct.addr, key, res) }()

	if !acct.dirtyStorage {
		res = common.Hash{}
		return
	}
	val, ok := acct.knownStorage[key]
	if !ok {
		if acct.interpretVersion() == beingDeleted {
			// When getting state on an account being deleted, it needs to get the committed state at version-1.
			val = acct.parent.GetStorageByVersion(acct.addr, acct.version-1, key, true)
		} else {
			val = acct.parent.GetStorageByVersion(acct.addr, acct.version, key, true)
		}
		acct.knownStorage[key] = val
	}
	res = val
	return
}

// Create attempts to create this non-existence account.
func (acct *mutableAccountImpl) Create() (revert func()) {
	log.Debugf("Account %v - Create() [Before %v-%v-%v-%v-%v]", acct.addr, acct.nonce, acct.balance, acct.codeHash, acct.dirtyStorage, acct.version)
	defer func() {
		log.Debugf("Account %v - Create() returns void [After %v-%v-%v-%v-%v]", acct.addr, acct.nonce, acct.balance, acct.codeHash, acct.dirtyStorage, acct.version)
	}()

	if acct.interpretVersion() == beingDeleted {
		log.Panicf("Cannot create to overwrite account being deleted")
	}
	if acct.interpretVersion() == updating {
		if !acct.Empty() {
			log.Panicf("Cannot create to overwrite non-empty account being updated")
		}
		// Note: create on an empty account in updating state is allowed.
		return func() {}
	}
	acct.version++
	return func() {
		acct.version--
	}
}

// SetNonce attempts to set the nonce of this account.
func (acct *mutableAccountImpl) SetNonce(nonce uint64) (revert func()) {
	log.Debugf("Account %v - SetNonce(%v) [Before %v-%v-%v-%v-%v]", acct.addr, nonce, acct.nonce, acct.balance, acct.codeHash, acct.dirtyStorage, acct.version)
	defer func() {
		log.Debugf("Account %v - SetNonce(%v) returns void [After %v-%v-%v-%v-%v]", acct.addr, nonce, acct.nonce, acct.balance, acct.codeHash, acct.dirtyStorage, acct.version)
	}()

	optionalRevert := func() {}
	if acct.interpretVersion() == notExisted {
		acct.version++
		optionalRevert = func() { acct.version-- }
	} else if acct.interpretVersion() == beingDeleted {
		if nonce != 0 {
			log.Panicf("Cannot modify account being deleted to have nonce %v", nonce)
		}
	}
	originalNonce := acct.nonce
	acct.nonce = nonce
	if originalNonce == acct.nonce {
		return optionalRevert
	}
	return func() {
		optionalRevert()
		acct.nonce = originalNonce
	}
}

// SetBalance attempts to set the balance of this account.
func (acct *mutableAccountImpl) SetBalance(balance *uint256.Int) (revert func()) {
	log.Debugf("Account %v - SetBalance(%v) [Before %v-%v-%v-%v-%v]", acct.addr, balance, acct.nonce, acct.balance, acct.codeHash, acct.dirtyStorage, acct.version)
	defer func() {
		log.Debugf("Account %v - SetBalance(%v) returns void [After %v-%v-%v-%v-%v]", acct.addr, balance, acct.nonce, acct.balance, acct.codeHash, acct.dirtyStorage, acct.version)
	}()

	optionalRevert := func() {}
	if acct.interpretVersion() == notExisted {
		acct.version++
		optionalRevert = func() { acct.version-- }
	}
	// Note: Set balance to a destructed account is allowed.

	originalBal := uint256.NewInt(0).Set(acct.balance)
	acct.balance.Set(balance)
	if originalBal.Cmp(acct.balance) == 0 {
		return optionalRevert
	}
	return func() {
		optionalRevert()
		acct.balance.Set(originalBal)
	}
}

// SetCode attempts to set the code of this account.
func (acct *mutableAccountImpl) SetCode(code []byte) (revert func()) {
	log.Debugf("Account %v - SetCode(%v) [Before %v-%v-%v-%v-%v]", acct.addr, len(code), acct.nonce, acct.balance, acct.codeHash, acct.dirtyStorage, acct.version)
	defer func() {
		log.Debugf("Account %v - SetCode(%v) returns void [After %v-%v-%v-%v-%v]", acct.addr, len(code), acct.nonce, acct.balance, acct.codeHash, acct.dirtyStorage, acct.version)
	}()

	optionalRevert := func() {}
	if acct.interpretVersion() == notExisted {
		acct.version++
		optionalRevert = func() { acct.version-- }
	}
	// Note: Set code to a destructed account is allowed.

	// Set code hash
	originalCodeHash := acct.codeHash
	acct.codeHash = crypto.Keccak256Hash(code)
	if originalCodeHash.Cmp(acct.codeHash) == 0 {
		return optionalRevert
	}

	// A code change, need to update known code diff
	optionalRevert2 := func() {}
	if originalCodeHash == types.EmptyCodeHash {
		// Code added
		acct.knownCodeDiff[acct.codeHash]++
		optionalRevert2 = func() {
			acct.knownCodeDiff[acct.codeHash]--
		}
	} else if acct.codeHash == types.EmptyCodeHash {
		// Code removed
		acct.knownCodeDiff[originalCodeHash]--
		optionalRevert2 = func() {
			acct.knownCodeDiff[acct.codeHash]++
		}
	} else {
		// This should never happen.
		log.Panicf("Cannot overwrite non empty code, current %v, new %v", originalCodeHash, acct.codeHash)
	}

	// Set known code
	var originalCode []byte
	if acct.knownCode != nil {
		originalCode = append([]byte(nil), acct.knownCode...)
	}
	acct.knownCode = code
	return func() {
		optionalRevert()
		optionalRevert2()
		acct.codeHash = originalCodeHash
		acct.knownCode = originalCode
	}
}

// SetState attempts to set the storage slot value of this account.
func (acct *mutableAccountImpl) SetState(key common.Hash, val common.Hash) (revert func()) {
	log.Debugf("Account %v - SetState(%v,%v) [Before %v-%v-%v-%v-%v]", acct.addr, key, val, acct.nonce, acct.balance, acct.codeHash, acct.dirtyStorage, acct.version)
	defer func() {
		log.Debugf("Account %v - SetState(%v,%v) returns void [After %v-%v-%v-%v-%v]", acct.addr, key, val, acct.nonce, acct.balance, acct.codeHash, acct.dirtyStorage, acct.version)
	}()

	optionalRevert := func() {}
	if acct.interpretVersion() == notExisted {
		acct.version++
		optionalRevert = func() { acct.version-- }
	}
	// Note: Set state to a destructed account is allowed.

	if acct.dirtyStorage {
		originalVal, ok := acct.knownStorage[key]
		acct.knownStorage[key] = val
		if !ok {
			return func() {
				optionalRevert()
				delete(acct.knownStorage, key)
			}
		}
		if originalVal.Cmp(val) == 0 {
			return optionalRevert
		}
		return func() {
			optionalRevert()
			acct.knownStorage[key] = originalVal
		}
	}
	acct.dirtyStorage = true
	acct.knownStorage[key] = val
	return func() {
		optionalRevert()
		acct.dirtyStorage = false
		delete(acct.knownStorage, key)
	}
}

// HasSelfDestructed checks if this account has been self destructed.
func (acct *mutableAccountImpl) HasSelfDestructed() (res bool) {
	log.Debugf("Account %v - HasSelfDestructed()", acct.addr)
	defer func() { log.Debugf("Account %v - HasSelfDestructed() returns %v", acct.addr, res) }()

	res = acct.interpretVersion() == beingDeleted
	return
}

// SelfDestruct attempts to destruct this account.
func (acct *mutableAccountImpl) SelfDestruct() (revert func()) {
	log.Debugf("Account %v - SelfDestruct() [Before %v-%v-%v-%v-%v]", acct.addr, acct.nonce, acct.balance, acct.codeHash, acct.dirtyStorage, acct.version)
	defer func() {
		log.Debugf("Account %v - SelfDestruct() returns void [After %v-%v-%v-%v-%v]", acct.addr, acct.nonce, acct.balance, acct.codeHash, acct.dirtyStorage, acct.version)
	}()

	if acct.interpretVersion() == beingDeleted {
		// Account can be destructed again with no effect on the account state.
		return func() {}
	}
	if acct.interpretVersion() == notExisted {
		log.Panic("cannot destruct not existed account")
	}
	// Self destruct needs to clear account balance.
	originalBal := uint256.NewInt(0).Set(acct.balance)
	acct.balance.Set(uint256.NewInt(0))
	acct.version++
	return func() {
		acct.version--
		acct.balance.Set(originalBal)
	}
}

// GetFinalized gets the finalized state after all state mutation.
func (acct *mutableAccountImpl) GetFinalized() (
	finalState itypes.AccountValue,
	knownCode []byte,
	knownCodeDiff map[common.Hash]int64,
	knownStorage map[common.Hash]common.Hash) {
	if acct.interpretVersion() == beingDeleted {
		if acct.codeHash != types.EmptyCodeHash {
			// update known code diff
			acct.knownCodeDiff[acct.codeHash]--
		}
		finalState = itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      acct.version + 1,
		}
		knownCode = nil
		knownCodeDiff = acct.knownCodeDiff
		knownStorage = map[common.Hash]common.Hash{}
		return
	}
	finalState = itypes.AccountValue{
		Nonce:        acct.nonce,
		Balance:      acct.balance,
		CodeHash:     acct.codeHash,
		DirtyStorage: acct.dirtyStorage,
		Version:      acct.version,
	}
	knownCode = acct.knownCode
	knownCodeDiff = acct.knownCodeDiff
	knownStorage = acct.knownStorage
	return
}

// Copy creates a deep copy of this account.
func (acct *mutableAccountImpl) Copy() mutableAccount {
	var knownCode []byte
	if acct.knownCode != nil {
		knownCode = append([]byte(nil), acct.knownCode...)
	}
	knownCodeDiff := make(map[common.Hash]int64)
	for k, v := range acct.knownCodeDiff {
		knownCodeDiff[k] = v
	}
	knownStorage := make(map[common.Hash]common.Hash)
	for k, v := range acct.knownStorage {
		knownStorage[k] = v
	}
	return &mutableAccountImpl{
		parent:        acct.parent,
		addr:          acct.addr,
		nonce:         acct.nonce,
		balance:       uint256.NewInt(0).Set(acct.balance),
		codeHash:      acct.codeHash,
		dirtyStorage:  acct.dirtyStorage,
		version:       acct.version,
		knownCode:     knownCode,
		knownCodeDiff: knownCodeDiff,
		knownStorage:  knownStorage,
	}
}

// Version gets the current version of the account.
func (acct *mutableAccountImpl) Version() uint64 {
	return acct.version
}
