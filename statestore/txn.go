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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb"
	lerrors "github.com/syndtr/goleveldb/leveldb/errors"
	itypes "github.com/wcgcyx/teler/types"
)

// transactionImpl implements Transaction.
type transactionImpl struct {
	ds  *leveldb.DB
	txn *leveldb.Transaction
}

// NewTransaction creates a new transaction to write.
func (s *stateStoreImpl) NewTransaction() (Transaction, error) {
	txn, err := s.ds.OpenTransaction()
	if err != nil {
		return nil, err
	}
	return &transactionImpl{ds: s.ds, txn: txn}, nil
}

// Persist layer log by applying all the changes.
// It includes update persisted height, root hash and commit all changes.
func (t *transactionImpl) PersistLayerLog(layerLog itypes.LayerLog) error {
	err := t.txn.Put(persistedHeightKey(), encodePersistedHeight(layerLog.BlockNumber, layerLog.RootHash), nil)
	if err != nil {
		return err
	}

	for addr, acctp := range layerLog.UpdatedAccounts {
		if acctp.Version == 0 {
			// Skip this account, this account does not exist before this layer
			// and has not been updated in this layer.
			continue
		} else if acctp.Version%3 == 0 {
			// This account has been deleted.
			// Delete account value
			err = t.txn.Delete(getAccountValueKey(addr), nil)
			if err != nil {
				return err
			}
			// Add account version
			err = t.txn.Put(getAccountVersionKey(addr), encodeAccountVersion(acctp.Version), nil)
			if err != nil {
				return err
			}
		} else if acctp.Version%3 == 2 {
			// This should never happen
			log.Panicf("attempt save account that is being deleted: %v", acctp)
		} else {
			// This account has been updated.
			// Update account value
			err = t.txn.Put(getAccountValueKey(addr), encodeAccountValue(&acctp), nil)
			if err != nil {
				return err
			}
			// Delete account version if any
			err = t.txn.Delete(getAccountVersionKey(addr), nil)
			if err != nil {
				return err
			}
		}
		// Check committed account version for possible GC
		version := uint64(0)
		data, err := t.ds.Get(getAccountVersionKey(addr), nil)
		if err == nil {
			version = decodeAccountVersion(data)
		} else if !errors.Is(err, lerrors.ErrNotFound) {
			return err
		} else {
			data, err = t.ds.Get(getAccountValueKey(addr), nil)
			if err == nil {
				acct, err := decodeAccountValue(data)
				if err != nil {
					return err
				}
				version = acct.Version
			} else if !errors.Is(err, lerrors.ErrNotFound) {
				return err
			}
		}
		if version > 0 {
			for i := version; i < acctp.Version; i++ {
				// Notify GC to clear old versions
				err = t.txn.Put(getGCKey(addr, i), []byte{}, nil)
				if err != nil {
					return err
				}
			}
		}
	}

	for codeHash, diff := range layerLog.UpdatedCodeCount {
		if diff == 0 {
			// This code hash require no update, skip.
			continue
		}
		// Get existing count for given codeHash
		val, err := t.ds.Get(getCodeKey(codeHash), nil)
		if err != nil {
			if !errors.Is(err, lerrors.ErrNotFound) {
				return err
			}
			// This code does not exist for given hash
			// Diff must not be negative
			if diff < 0 {
				return fmt.Errorf("code %v does not exist locally but require diff of %v", codeHash, diff)
			}
			// This code must present in the code preimage
			code, ok := layerLog.CodePreimage[codeHash]
			if !ok {
				return fmt.Errorf("code %v does not exist locally and in the code preimage", codeHash)
			}
			err = t.txn.Put(getCodeKey(codeHash), encodeCodeValue(code, diff), nil)
			if err != nil {
				return err
			}
			continue
		}
		// This code exists for given hash locally
		code, existing, err := decodeCodeValue(val)
		if err != nil {
			return err
		}
		new := existing + diff
		if new < 0 {
			// This should never happen.
			return fmt.Errorf("code %v exist locally with count %v, but need to require at least diff of %v", codeHash, existing, diff)
		} else if new == 0 {
			// Need to remove code
			err = t.txn.Delete(getCodeKey(codeHash), nil)
			if err != nil {
				return err
			}
		} else {
			// Need to put new count for given code
			err = t.txn.Put(getCodeKey(codeHash), encodeCodeValue(code, new), nil)
			if err != nil {
				return err
			}
		}
	}

	for storageKey, storage := range layerLog.UpdatedStorage {
		addr, version := itypes.SplitAccountStorageKey(storageKey)
		for k, v := range storage {
			err := t.txn.Put(getStorageKey(addr, version, k), encodeStorage(v), nil)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// PutChildren puts children to the state with the given root hash.
func (t *transactionImpl) PutChildren(height uint64, rootHash common.Hash, children []common.Hash) error {
	return t.txn.Put(getChildrenKey(height, rootHash), encodeChildren(children), nil)
}

// DeleteChildren deletes children of the state with the given root hash.
func (t *transactionImpl) DeleteChildren(height uint64, rootHash common.Hash) error {
	return t.txn.Delete(getChildrenKey(height, rootHash), nil)
}

// PutLayerLog puts layer log to the state with the given root hash.
func (t *transactionImpl) PutLayerLog(layerLog itypes.LayerLog) error {
	return t.txn.Put(getLayerLogKey(layerLog.BlockNumber, layerLog.RootHash), encodeLayerLog(layerLog), nil)
}

// DeleteLayerLog deletes layer log of the state with the given root hash.
func (t *transactionImpl) DeleteLayerLog(height uint64, rootHash common.Hash) error {
	return t.txn.Delete(getLayerLogKey(height, rootHash), nil)
}

// Commit commits all changes.
func (t *transactionImpl) Commit() error {
	return t.txn.Commit()
}

// Discard discards all changes.
func (t *transactionImpl) Discard() {
	t.txn.Discard()
}
