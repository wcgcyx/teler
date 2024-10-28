package types

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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// AccountValue is used to represent the state of an account.
type AccountValue struct {
	// The nonce of the account
	Nonce uint64

	// The balance of the account
	Balance *uint256.Int

	// The code hash of this account
	CodeHash common.Hash

	// Flag indicating if this account's storage trie is empty
	DirtyStorage bool

	// Version of the account
	Version uint64
}

// LayerLog is used to represent the state existed in a state layer.
type LayerLog struct {
	// Block number of this layer
	BlockNumber uint64

	// Root hash of this layer
	RootHash common.Hash

	// Updated accounts in this layer
	// A map from addr -> acct value
	UpdatedAccounts map[common.Address]AccountValue

	// Updated code (added/deleted) count in this layer
	// A map from codeHash -> change in count.
	UpdatedCodeCount map[common.Hash]int64

	// Seen code in this layer
	// A map from codeHash -> code
	CodePreimage map[common.Hash][]byte

	// Updated storage in this layer
	// A map from "addr.Hex()-version" -> key -> value
	UpdatedStorage map[string]map[common.Hash]common.Hash
}

// Creates layerlog for the genesis block.
func LayerLogFromGenesis(genesis *core.Genesis, genesisRoot common.Hash) *LayerLog {
	layerLog := &LayerLog{
		BlockNumber:      0,
		RootHash:         genesisRoot,
		UpdatedAccounts:  make(map[common.Address]AccountValue),
		UpdatedCodeCount: make(map[common.Hash]int64),
		CodePreimage:     make(map[common.Hash][]byte),
		UpdatedStorage:   make(map[string]map[common.Hash]common.Hash),
	}
	for addr, account := range genesis.Alloc {
		acctp := AccountValue{
			Nonce:        account.Nonce,
			Balance:      uint256.MustFromBig(account.Balance),
			CodeHash:     crypto.Keccak256Hash(account.Code),
			DirtyStorage: false,
			Version:      1,
		}
		if len(account.Code) > 0 {
			layerLog.UpdatedCodeCount[acctp.CodeHash]++
			layerLog.CodePreimage[acctp.CodeHash] = account.Code
		}
		if len(account.Storage) > 0 {
			acctp.DirtyStorage = true
			layerLog.UpdatedStorage[GetAccountStorageKey(addr, 1)] = make(map[common.Hash]common.Hash)
			for k, v := range account.Storage {
				layerLog.UpdatedStorage[GetAccountStorageKey(addr, 1)][k] = v
			}
		}
		layerLog.UpdatedAccounts[addr] = acctp
	}
	return layerLog
}
