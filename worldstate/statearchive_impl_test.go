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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/teler/statestore"
	"go.uber.org/mock/gomock"
)

func TestNewStateArchive(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := statestore.NewMockStateStore(ctrl)
	m.
		EXPECT().
		GetPersistedHeight().
		Return(uint64(0), common.HexToHash("0x1"), nil)
	m.
		EXPECT().
		GetChildren(uint64(0), common.HexToHash("0x1")).
		Return([]common.Hash{}, nil)

	genesis := core.DefaultGenesisBlock()

	archive, err := NewLayeredWorldStateArchiveImpl(Opts{}, genesis.Config, m)
	assert.Nil(t, err)
	assert.NotNil(t, archive)
}

func TestStateArchiveGetters(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := statestore.NewMockStateStore(ctrl)
	m.
		EXPECT().
		GetPersistedHeight().
		Return(uint64(0), common.HexToHash("0x1"), nil)
	m.
		EXPECT().
		GetChildren(uint64(0), common.HexToHash("0x1")).
		Return([]common.Hash{}, nil)

	genesis := core.DefaultGenesisBlock()

	archive, err := NewLayeredWorldStateArchiveImpl(Opts{
		MaxLayerToRetain: 256,
		PruningFrequency: 128,
	}, genesis.Config, m)
	assert.Nil(t, err)

	assert.Equal(t, genesis.Config, archive.GetChainConfig())
	assert.Equal(t, uint64(256), archive.GetMaxLayerToRetain())
	assert.Equal(t, uint64(128), archive.GetPruningFrequency())
}

func TestStateArchiveStateRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := statestore.NewMockStateStore(ctrl)
	m.
		EXPECT().
		GetPersistedHeight().
		Return(uint64(0), common.HexToHash("0x1"), nil)
	m.
		EXPECT().
		GetChildren(uint64(0), common.HexToHash("0x1")).
		Return([]common.Hash{}, nil)

	genesis := core.DefaultGenesisBlock()

	archive, err := NewLayeredWorldStateArchiveImpl(Opts{
		MaxLayerToRetain: 256,
		PruningFrequency: 128,
	}, genesis.Config, m)
	assert.Nil(t, err)

	archive.Deregister(2, common.HexToHash("0x2"))
	assert.False(t, archive.Has(2, common.HexToHash("0x2")))
	archive.Register(2, common.HexToHash("0x2"), nil)
	assert.True(t, archive.Has(2, common.HexToHash("0x2")))
	archive.Deregister(2, common.HexToHash("0x2"))
	assert.False(t, archive.Has(2, common.HexToHash("0x2")))
}

func TestStateArchiveGetLayered(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := statestore.NewMockStateStore(ctrl)
	m.
		EXPECT().
		GetPersistedHeight().
		Return(uint64(0), common.HexToHash("0x1"), nil)
	m.
		EXPECT().
		GetChildren(uint64(0), common.HexToHash("0x1")).
		Return([]common.Hash{}, nil)

	genesis := core.DefaultGenesisBlock()

	archive, err := NewLayeredWorldStateArchiveImpl(Opts{
		MaxLayerToRetain: 256,
		PruningFrequency: 128,
	}, genesis.Config, m)
	assert.Nil(t, err)

	_, err = archive.GetLayared(1, common.HexToHash("0x1"))
	assert.NotNil(t, err)

	_, err = archive.GetLayared(0, common.HexToHash("0x2"))
	assert.NotNil(t, err)

	layered, err := archive.GetLayared(0, common.HexToHash("0x1"))
	assert.Nil(t, err)
	assert.NotNil(t, layered)
}

func TestStateArchiveGetMutable(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := statestore.NewMockStateStore(ctrl)
	m.
		EXPECT().
		GetPersistedHeight().
		Return(uint64(0), common.HexToHash("0x1"), nil)
	m.
		EXPECT().
		GetChildren(uint64(0), common.HexToHash("0x1")).
		Return([]common.Hash{}, nil)

	genesis := core.DefaultGenesisBlock()

	archive, err := NewLayeredWorldStateArchiveImpl(Opts{
		MaxLayerToRetain: 256,
		PruningFrequency: 128,
	}, genesis.Config, m)
	assert.Nil(t, err)

	_, err = archive.GetMutable(1, common.HexToHash("0x1"))
	assert.NotNil(t, err)

	_, err = archive.GetMutable(0, common.HexToHash("0x2"))
	assert.NotNil(t, err)

	layered, err := archive.GetMutable(0, common.HexToHash("0x1"))
	assert.Nil(t, err)
	assert.NotNil(t, layered)
}
