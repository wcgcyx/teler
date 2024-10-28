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
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
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
				tempRes, err := s.ds.Query(s.routineCtx, query.Query{
					Prefix:   gcKey,
					KeysOnly: true,
				})
				if err != nil {
					log.Warnf("GC - Fail to query ds: %v", err.Error())
					return
				}
				defer tempRes.Close()
				for {
					if s.routineCtx.Err() != nil {
						log.Warnf("Exit GC round due to context cancelled: %v", s.routineCtx.Err().Error())
						return
					}
					e, ok := tempRes.NextSync()
					if !ok {
						break
					}
					if e.Error != nil {
						log.Warnf("GC - Fail to get next entry: %v", e.Error.Error())
						return
					}
					if len(e.Key) <= 1 {
						log.Warnf("GC - Key smaller than 1: %v", e.Key)
						continue
					}
					addr, version := splitGCKey(e.Key[1:])
					clearedSlots := 0
					// Clear all storages involved with this version
					txn, err := s.ds.Batch(s.routineCtx)
					if err != nil {
						log.Warnf("GC - Fail to create batch to clear %v-%v: %v", addr, version, err.Error())
						continue
					}
					err = func() error {
						addrStr := base64.URLEncoding.EncodeToString(addr.Bytes())
						tempRes2, err2 := s.ds.Query(s.routineCtx, query.Query{
							Prefix:   storageKey + separator + addrStr + separator + strconv.FormatUint(version, 10),
							KeysOnly: true,
						})
						if err2 != nil {
							return err2
						}
						defer tempRes2.Close()
						for {
							if s.routineCtx.Err() != nil {
								return s.routineCtx.Err()
							}
							e2, ok2 := tempRes2.NextSync()
							if !ok2 {
								return nil
							}
							if e2.Error != nil {
								return e2.Error
							}
							if len(e2.Key) <= 1 {
								return fmt.Errorf("key smaller than 1: %v", e.Key)
							}
							_, _, k := splitStorageKey(e2.Key[1:])
							err = txn.Delete(s.routineCtx, getStorageKey(addr, version, k))
							if err != nil {
								return err
							}
							clearedSlots++
						}
					}()
					if err != nil {
						log.Warnf("GC - Fail to clear %v-%v: %v", addr, version, err.Error())
						continue
					}
					err = txn.Delete(s.routineCtx, datastore.NewKey(e.Key))
					if err != nil {
						log.Warnf("GC - Fail to clear gc entry %v-%v: %v", addr, version, err.Error())
						continue
					}
					err = txn.Commit(s.routineCtx)
					if err != nil {
						log.Warnf("GC - Fail to commit to clear %v-%v: %v", addr, version, err.Error())
						continue
					}
					totalCleanedAccts++
					totalCleanedSlots += clearedSlots
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
