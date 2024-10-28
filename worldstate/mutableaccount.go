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
	"github.com/holiman/uint256"
	itypes "github.com/wcgcyx/teler/types"
)

// mutableAccount is an abstract type built on top of layered state and layered log
// It supports state mutation that can be used to compute a state transition.
type mutableAccount interface {
	// Empty returns if this account is empty.
	Empty() bool

	// Exist returns if this account exists.
	Exists(isEIP158 bool) bool

	// Nonce returns the nonce of this account.
	Nonce() uint64

	// Balance returns the balance of this account.
	Balance() *uint256.Int

	// CodeHash returns the code hash of this account.
	CodeHash() common.Hash

	// Code returns the code of this account.
	Code() []byte

	// DirtyStorage returns if account's storage trie is empty.
	DirtyStorage() bool

	// GetState returns the given storage slot value of this account.
	GetState(key common.Hash) common.Hash

	// Create attempts to create this non-existence account.
	Create() (revert func())

	// SetNonce attempts to set the nonce of this account.
	SetNonce(nonce uint64) (revert func())

	// SetBalance attempts to set the balance of this account.
	SetBalance(balance *uint256.Int) (revert func())

	// SetCode attempts to set the code of this account.
	SetCode(code []byte) (revert func())

	// SetState attempts to set the storage slot value of this account.
	SetState(key common.Hash, val common.Hash) (revert func())

	// HasSelfDestructed checks if this account has been self destructed.
	HasSelfDestructed() bool

	// SelfDestruct attempts to destruct this account.
	SelfDestruct() (revert func())

	// GetFinalized gets the finalized state after all state mutation.
	GetFinalized() (finalState itypes.AccountValue, knownCode []byte, knownCodeDiff map[common.Hash]int64, knownStorage map[common.Hash]common.Hash)

	// Copy creates a deep copy of this account.
	Copy() mutableAccount

	// Version gets the current version of the account.
	Version() uint64
}
