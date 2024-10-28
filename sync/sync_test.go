package sync

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
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/teler/backend"
	"github.com/wcgcyx/teler/blockchain"
	gomock "go.uber.org/mock/gomock"
)

func TestForwardSync(t *testing.T) {
	ctrl := gomock.NewController(t)

	m1 := backend.NewMockBackend(ctrl)
	m2 := NewMockBlockSource(ctrl)
	m3 := blockchain.NewMockBlockchain(ctrl)

	// Forward sync from blk 900 to 1000
	m1.
		EXPECT().
		Blockchain().
		Return(m3)
	prvBlk := types.NewBlockWithHeader(&types.Header{
		Number: big.NewInt(900),
	})
	m3.EXPECT().GetHead(gomock.Any()).Return(prvBlk, nil)

	for i := int64(0); i < 100; i++ {
		blk := types.NewBlockWithHeader(&types.Header{
			Number: big.NewInt(901 + i),
		})
		m2.
			EXPECT().
			BlockByNumber(gomock.Any(), big.NewInt(901+i)).
			Return(blk, nil)
		m1.
			EXPECT().
			ImportBlock(gomock.Any(), blk, prvBlk).
			Return(nil)
		prvBlk = blk
	}
	assert.Nil(t, ForwardSync(context.Background(), m1, m2, 1000))
}

func TestBackwardSync(t *testing.T) {
	ctrl := gomock.NewController(t)

	m1 := backend.NewMockBackend(ctrl)
	m2 := NewMockBlockSource(ctrl)
	m3 := blockchain.NewMockBlockchain(ctrl)

	// Backward sync
	blk := types.NewBlockWithHeader(&types.Header{
		ParentHash: common.HexToHash(strconv.FormatUint(999, 16)),
	})
	for i := uint64(0); i < 100; i++ {
		m3.
			EXPECT().
			HasBlock(gomock.Any(), common.HexToHash(strconv.FormatUint(999-i, 16))).
			Return(false, nil)
		m2.
			EXPECT().
			BlockByHash(gomock.Any(), common.HexToHash(strconv.FormatUint(999-i, 16))).
			Return(types.NewBlockWithHeader(&types.Header{
				ParentHash: common.HexToHash(strconv.FormatUint(998-i, 16)),
			}), nil)
	}
	m3.
		EXPECT().
		HasBlock(gomock.Any(), common.HexToHash(strconv.FormatUint(899, 16))).
		Return(true, nil)
	m1.
		EXPECT().
		Blockchain().
		Return(m3).AnyTimes()
	m1.
		EXPECT().
		ImportBlocks(gomock.Any(), gomock.Any()).
		Return(nil)
	assert.Nil(t, BackwardSync(context.Background(), m1, m2, blk, 128))
}

func TestBackwardSyncExceedingMaxBlksToQuery(t *testing.T) {
	ctrl := gomock.NewController(t)

	m1 := backend.NewMockBackend(ctrl)
	m2 := NewMockBlockSource(ctrl)
	m3 := blockchain.NewMockBlockchain(ctrl)

	// Backward sync
	blk := types.NewBlockWithHeader(&types.Header{
		ParentHash: common.HexToHash(strconv.FormatUint(999, 16)),
	})
	for i := uint64(0); i < 100; i++ {
		m3.
			EXPECT().
			HasBlock(gomock.Any(), common.HexToHash(strconv.FormatUint(999-i, 16))).
			Return(false, nil)
		m2.
			EXPECT().
			BlockByHash(gomock.Any(), common.HexToHash(strconv.FormatUint(999-i, 16))).
			Return(types.NewBlockWithHeader(&types.Header{
				ParentHash: common.HexToHash(strconv.FormatUint(998-i, 16)),
			}), nil)
	}
	m3.
		EXPECT().
		HasBlock(gomock.Any(), common.HexToHash(strconv.FormatUint(899, 16))).
		Return(false, nil)
	m1.
		EXPECT().
		Blockchain().
		Return(m3).AnyTimes()
	assert.NotNil(t, BackwardSync(context.Background(), m1, m2, blk, 100))
}
