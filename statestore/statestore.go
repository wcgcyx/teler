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
	"github.com/ethereum/go-ethereum/common"
	itypes "github.com/wcgcyx/teler/types"
)

type StateStore interface {
	// GetPersistedHeight gets the persisted state height and root.
	GetPersistedHeight() (uint64, common.Hash, error)

	// GetChildren gets the child states for a state with the given root hash.
	GetChildren(height uint64, rootHash common.Hash) ([]common.Hash, error)

	// GetLayerLog gets the layer log for a state with the given root hash.
	GetLayerLog(height uint64, rootHash common.Hash) (itypes.LayerLog, error)

	// GetAccountValue gets the persisted account value for given address.
	GetAccountValue(addr common.Address) (itypes.AccountValue, error)

	// GetStorageByVersion gets the persisted storage value for given key.
	GetStorageByVersion(addr common.Address, version uint64, key common.Hash) (common.Hash, error)

	// GetCodeByHash gets the persisted code for given hash.
	GetCodeByHash(codeHash common.Hash) ([]byte, error)

	// NewTransaction creates a new transaction to write.
	NewTransaction() (Transaction, error)

	// Shutdown safely shuts the statestore down.
	Shutdown()
}

type Transaction interface {
	// Persist layer log by applying all the changes.
	// It includes update persisted height, root hash and commit all changes.
	PersistLayerLog(layerLog itypes.LayerLog) error

	// PutChildren puts children to the state with the given root hash.
	PutChildren(height uint64, rootHash common.Hash, children []common.Hash) error

	// DeleteChildren deletes children of the state with the given root hash.
	DeleteChildren(height uint64, rootHash common.Hash) error

	// PutLayerLog puts layer log to the state with the given root hash.
	PutLayerLog(layerLog itypes.LayerLog) error

	// DeleteLayerLog deletes layer log of the state with the given root hash.
	DeleteLayerLog(height uint64, rootHash common.Hash) error

	// Commit commits all changes.
	Commit() error

	// Discard discards all changes.
	Discard()
}
