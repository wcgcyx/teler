package node

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
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	logging "github.com/ipfs/go-log"
	"github.com/wcgcyx/teler/backend"
	isync "github.com/wcgcyx/teler/sync"
)

// Logger
var log = logging.Logger("node")

type Node struct {
	opts Opts

	// Trusted block source
	BlkSrc isync.BlockSource

	// Backend
	Backend backend.Backend

	// Process related
	routineCtx context.Context
	exitLoop   chan bool

	// Pausing related
	paused     bool
	pausedLock sync.RWMutex
	pause      func()

	// Shutdown function
	shutdown func()
}

// NewNode creates the main node.
func NewNode(
	opts Opts,
	blkSrc isync.BlockSource,
	b backend.Backend,
) (*Node, error) {
	routineCtx, cancel := context.WithCancel(context.Background())
	node := &Node{
		opts:       opts,
		BlkSrc:     blkSrc,
		Backend:    b,
		routineCtx: routineCtx,
		exitLoop:   make(chan bool),
		paused:     false,
		pausedLock: sync.RWMutex{},
		shutdown:   func() { cancel() },
	}
	return node, nil
}

// The mainloop of node.
func (node *Node) Mainloop() {
	defer func() {
		node.exitLoop <- true
	}()
	log.Infof("Start main routine...")

	after := time.NewTicker(node.opts.CheckFrequency)
	for ; true; <-after.C {
		select {
		case <-node.routineCtx.Done():
			log.Infof("Shutdown node mainloop")
			return
		default:
			if node.paused {
				log.Warnf("Mainloop is paued, waiting to resume...")
				continue
			}
			localHead, err := node.Backend.Blockchain().GetHead(node.routineCtx)
			if err != nil {
				log.Errorf("Fail to obtain local head: %v", err.Error())
				continue
			}
			remoteHead, err := node.BlkSrc.BlockByNumber(node.routineCtx, big.NewInt(rpc.LatestBlockNumber.Int64()))
			if err != nil {
				log.Warnf("Fail to obtain remote head: %v", err.Error())
				continue
			}
			remoteSafe, err := node.BlkSrc.HeaderByNumber(node.routineCtx, big.NewInt(rpc.SafeBlockNumber.Int64()))
			if err != nil {
				log.Warnf("Fail to obtain remote safe: %v", err.Error())
				continue
			}
			remoteFinalized, err := node.BlkSrc.HeaderByNumber(node.routineCtx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
			if err != nil {
				log.Warnf("Fail to obtain remote finalized: %v", err.Error())
				continue
			}
			local := localHead.NumberU64()
			remote := remoteHead.NumberU64()
			if remote <= local {
				diff := local - remote
				if diff > node.opts.MaxAllowedLeadBlocks {
					log.Errorf("Remote chain is greatly behind local chain, local %v, remote %v", local, remote)
					continue
				}
				// Check if remote is known by local
				remoteHash := remoteHead.Hash()
				exists, err := node.Backend.Blockchain().HasBlock(node.routineCtx, remoteHash)
				if err != nil {
					log.Errorf("Fail to obtain block for %v: %v", remoteHash, err.Error())
					continue
				}
				if exists {
					log.Infof("Remote latest block %v has already been seen by local", remoteHash)
					continue
				}
				log.Infof("New head block %v detected, attempt backward sync...", remoteHash)
				err = isync.BackwardSync(node.syncContext(), node.Backend, node.BlkSrc, remoteHead, node.opts.BackwardSyncMaxBlocksToQueryReorg)
				if err != nil {
					log.Errorf("Fail to do backward sync (reorg) from %v to %v: %v", remoteHead, localHead, err.Error())
					continue
				}
				node.Backend.SetSafeTag(node.routineCtx, remoteSafe.Hash())
				node.Backend.SetFinalizedTag(node.routineCtx, remoteFinalized.Hash())
			} else {
				diff := remote - local
				if diff > node.opts.ForwardSyncMinDistanceToStart {
					err = isync.ForwardSync(node.syncContext(), node.Backend, node.BlkSrc, remote-node.opts.ForwardSyncTargetGap)
					if err != nil {
						log.Errorf("Fail to forward sync from %v to %v: %v", local+1, remote-node.opts.ForwardSyncTargetGap, err.Error())
						continue
					}
				} else {
					err = isync.BackwardSync(node.syncContext(), node.Backend, node.BlkSrc, remoteHead, node.opts.BackwardSyncMaxBlocksToQuery)
					if err != nil {
						log.Errorf("Fail to do backward sync from %v to %v: %v", remoteHead, localHead, err.Error())
						continue
					}
				}
			}
		}
		if node.routineCtx.Err() != nil {
			log.Warnf("Exit mainloop due to context cancelled: %v", node.routineCtx.Err().Error())
			return
		}
	}
}

// syncContext is used to obtian the context for sync process.
func (node *Node) syncContext() context.Context {
	node.pausedLock.Lock()
	defer node.pausedLock.Unlock()
	ctx, cancel := context.WithCancel(node.routineCtx)
	node.pause = cancel
	return ctx
}

// Pause pauses any active sync process.
func (node *Node) Pause() {
	node.pausedLock.Lock()
	defer node.pausedLock.Unlock()
	node.paused = true
	node.pause()
	node.pause = func() {}
}

// Unpause resumes sync.
func (node *Node) Unpause() {
	node.pausedLock.Lock()
	defer node.pausedLock.Unlock()
	node.paused = false
}

// Shutdown safely shuts down the main routine.
func (node *Node) Shutdown() {
	log.Infof("Close main routine...")
	node.shutdown()
	<-node.exitLoop
}
