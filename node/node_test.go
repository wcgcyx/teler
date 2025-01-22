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
	"fmt"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/teler/backend"
	"github.com/wcgcyx/teler/blockchain"
	"github.com/wcgcyx/teler/worldstate"
	"go.uber.org/mock/gomock"
)

func TestNormalSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	remote := &mockRemoteBlockchain{
		head: types.NewBlockWithHeader(&types.Header{
			Number:     big.NewInt(1000),
			ParentHash: common.HexToHash(strconv.FormatUint(999, 16)),
		}),
		ctx: ctx,
	}
	go remote.Loop()

	localChain := &mockLocalBlockchain{
		blks: []*types.Block{
			types.NewBlockWithHeader(&types.Header{
				Number:     big.NewInt(1),
				ParentHash: common.HexToHash("0x0"),
			}),
		},
		seen: make(map[common.Hash]bool),
	}

	ctrl := gomock.NewController(t)
	m1 := backend.NewMockBackend(ctrl)
	m2 := blockchain.NewMockBlockchain(ctrl)
	m3 := worldstate.NewMockLayeredWorldStateArchive(ctrl)
	m1.
		EXPECT().
		Blockchain().
		Return(m2).
		AnyTimes()
	m1.
		EXPECT().
		SetFinalizedTag(gomock.Any(), gomock.Any()).
		AnyTimes()
	m1.
		EXPECT().
		SetSafeTag(gomock.Any(), gomock.Any()).
		AnyTimes()
	m1.
		EXPECT().
		ImportBlock(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, blk *types.Block, _ *types.Block) error {
			return localChain.ImportBlk(blk)
		}).
		AnyTimes()
	m1.
		EXPECT().
		ImportBlocks(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, blks []*types.Block) error {
			for _, blk := range blks {
				if err := localChain.ImportBlk(blk); err != nil {
					return err
				}
			}
			return nil
		}).
		AnyTimes()
	m1.
		EXPECT().
		StateArchive().
		Return(m3).
		AnyTimes()
	m2.
		EXPECT().
		GetHead(gomock.Any()).
		DoAndReturn(func(context.Context) (*types.Block, error) {
			return localChain.GetHead(), nil
		}).
		AnyTimes()
	m2.
		EXPECT().
		HasBlock(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, hash common.Hash) (bool, error) {
			res := localChain.HasBlock(hash)
			return res, nil
		}).
		AnyTimes()
	m3.
		EXPECT().
		GetMaxLayerToRetain().
		Return(uint64(256))
	m3.
		EXPECT().
		GetPruningFrequency().
		Return(uint64(127))
	m3.
		EXPECT().
		SetMaxLayerToRetain(uint64(1))
	m3.
		EXPECT().
		SetPruningFrequency(uint64(1))
	m3.
		EXPECT().
		SetMaxLayerToRetain(uint64(256))
	m3.
		EXPECT().
		SetPruningFrequency(uint64(127))

	n, err := NewNode(Opts{
		CheckFrequency:                    600 * time.Millisecond,
		MaxAllowedLeadBlocks:              64,
		ForwardSyncMinDistanceToStart:     256,
		ForwardSyncTargetGap:              64,
		BackwardSyncMaxBlocksToQuery:      260,
		BackwardSyncMaxBlocksToQueryReorg: 64,
	}, remote, m1)
	assert.Nil(t, err)
	assert.NotNil(t, n)
	go n.Mainloop()
	defer n.Shutdown()

	time.Sleep(10 * time.Second)
	cancel()

	time.Sleep(time.Second)
	// Should catch up in the end
	assert.Equal(t, remote.head.NumberU64(), localChain.GetHead().NumberU64())
}

type mockRemoteBlockchain struct {
	head *types.Block
	ctx  context.Context
}

func (b *mockRemoteBlockchain) Loop() {
	after := time.NewTicker(time.Second)
	for ; true; <-after.C {
		select {
		case <-b.ctx.Done():
			return
		default:
			b.head = types.NewBlockWithHeader(&types.Header{
				Number:     big.NewInt(0).Add(b.head.Number(), big.NewInt(1)),
				ParentHash: common.HexToHash(strconv.FormatUint(b.head.Number().Uint64(), 16)),
			})
			// fmt.Printf("Block %v produced\n", b.head.Number())
		}
	}
}

func (b *mockRemoteBlockchain) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	number := big.NewInt(0)
	number.SetString(hash.Hex()[2:], 16)
	prv := big.NewInt(0).Sub(number, big.NewInt(1))
	return types.NewBlockWithHeader(&types.Header{
		Number:     number,
		ParentHash: common.HexToHash(strconv.FormatUint(prv.Uint64(), 16)),
	}), nil
}

func (b *mockRemoteBlockchain) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	if number.Cmp(big.NewInt(rpc.LatestBlockNumber.Int64())) == 0 ||
		number.Cmp(big.NewInt(rpc.SafeBlockNumber.Int64())) == 0 ||
		number.Cmp(big.NewInt(rpc.FinalizedBlockNumber.Int64())) == 0 {
		return b.head, nil
	}
	prv := big.NewInt(0).Sub(number, big.NewInt(1))
	return types.NewBlockWithHeader(&types.Header{
		Number:     number,
		ParentHash: common.HexToHash(strconv.FormatUint(prv.Uint64(), 16)),
	}), nil
}

func (b *mockRemoteBlockchain) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	prv := big.NewInt(0).Sub(number, big.NewInt(1))
	return &types.Header{
		Number:     number,
		ParentHash: common.HexToHash(strconv.FormatUint(prv.Uint64(), 16)),
	}, nil
}

func (b *mockRemoteBlockchain) Close() {
}

type mockLocalBlockchain struct {
	blks []*types.Block
	seen map[common.Hash]bool
}

func (b *mockLocalBlockchain) GetHead() *types.Block {
	return b.blks[len(b.blks)-1]
}

func (b *mockLocalBlockchain) HasBlock(hash common.Hash) bool {
	_, ok := b.seen[hash]
	if ok {
		return true
	}
	target, _ := big.NewInt(0).SetString(hash.Hex()[2:], 16)
	current := big.NewInt(b.GetHead().Number().Int64())
	return target.Cmp(current) <= 0
}

func (b *mockLocalBlockchain) ImportBlk(blk *types.Block) error {
	current := b.GetHead().NumberU64()
	target := blk.NumberU64()
	if target-current != 1 {
		return fmt.Errorf("fail to import, expect %v got %v\n", current+1, target)
	}
	b.blks = append(b.blks, blk)
	// fmt.Printf("Imported block %v\n", blk.NumberU64())
	b.seen[blk.Hash()] = true
	return nil
}
