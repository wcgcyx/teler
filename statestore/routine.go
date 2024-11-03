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
	"encoding/binary"
	"time"

	"github.com/syndtr/goleveldb/leveldb/util"
)

func (s *stateStoreImpl) gcRoutine() {
	defer func() {
		s.exitLoop <- true
	}()

	after := time.NewTicker(s.opts.GCPeriod)
	for {
		select {
		case <-s.routineCtx.Done():
			log.Infof("Exit GC routine")
			return
		case <-after.C:
			log.Infof("Start GC round")
			totalCleanedAccts := 0
			totalCleanedSlots := 0
			func() {
				iter := s.ds.NewIterator(util.BytesPrefix([]byte(gcKey)), nil)
				for iter.Next() {
					if s.routineCtx.Err() != nil {
						log.Warnf("Exit GC round due to context cancelled: %v", s.routineCtx.Err().Error())
						return
					}
					addr, version := splitGCKey(iter.Key())
					clearedSlots := 0
					// Clear all storages involved with this version
					txn, err := s.ds.OpenTransaction()
					if err != nil {
						log.Warnf("GC - Fail to create batch to clear %v-%v: %v", addr, version, err.Error())
						continue
					}
					defer txn.Discard()
					err = func() error {
						b := make([]byte, 8)
						binary.LittleEndian.PutUint64(b, version)
						prefix := append([]byte(storageKey+separator), addr.Bytes()...)
						prefix = append(prefix, []byte(separator)...)
						prefix = append(prefix, b...)
						prefix = append(prefix, []byte(separator)...)
						iter2 := s.ds.NewIterator(util.BytesPrefix(prefix), nil)
						for iter2.Next() {
							if s.routineCtx.Err() != nil {
								return s.routineCtx.Err()
							}
							_, _, k := splitStorageKey(iter2.Key())
							err = txn.Delete(getStorageKey(addr, version, k), nil)
							if err != nil {
								return err
							}
							clearedSlots++
						}
						iter2.Release()
						err := iter2.Error()
						if err != nil {
							return err
						}
						return nil
					}()
					if err != nil {
						log.Warnf("GC - Fail to clear %v-%v: %v", addr, version, err.Error())
						continue
					}
					err = txn.Delete(iter.Key(), nil)
					if err != nil {
						log.Warnf("GC - Fail to clear gc entry %v-%v: %v", addr, version, err.Error())
						continue
					}
					err = txn.Commit()
					if err != nil {
						log.Warnf("GC - Fail to commit to clear %v-%v: %v", addr, version, err.Error())
						continue
					}
					totalCleanedAccts++
					totalCleanedSlots += clearedSlots
				}
				iter.Release()
				err := iter.Error()
				if err != nil {
					log.Warnf("GC - Fail to query ds: %v", err.Error())
					return
				}
			}()
			log.Infof("GC round cleared %v accounts with %v storage slots", totalCleanedAccts, totalCleanedSlots)
		}
		if s.routineCtx.Err() != nil {
			log.Warnf("Exit mainloop due to context cancelled: %v", s.routineCtx.Err().Error())
			return
		}
	}
}
