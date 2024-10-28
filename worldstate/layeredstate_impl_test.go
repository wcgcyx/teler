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
	itypes "github.com/wcgcyx/teler/types"
	"go.uber.org/mock/gomock"
)

func TestNewLayeredState(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	m3 := NewMockLayeredWorldState(ctrl)

	m1.
		EXPECT().
		GetLayerLog(uint64(1), common.HexToHash("0x11")).
		Return(itypes.LayerLog{
			BlockNumber: 1,
		}, nil)
	m2.
		EXPECT().
		Register(uint64(1), common.HexToHash("0x11"), gomock.Any())
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x11")).
		Return([]common.Hash{}, nil)

	layer, err := newLayeredWorldState(m1, m2, m3, 1, common.HexToHash("0x11"))
	assert.Nil(t, err)
	assert.NotNil(t, layer)

	assert.Equal(t, uint64(1), layer.GetHeight())
}

func TestLayerGetAccountValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	m3 := NewMockLayeredWorldState(ctrl)

	m1.
		EXPECT().
		GetLayerLog(uint64(1), common.HexToHash("0x11")).
		Return(itypes.LayerLog{
			UpdatedAccounts: map[common.Address]itypes.AccountValue{
				common.HexToAddress(testAcctStr): {
					Nonce: 1,
				},
			},
		}, nil)
	m2.
		EXPECT().
		Register(uint64(1), common.HexToHash("0x11"), gomock.Any())
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x11")).
		Return([]common.Hash{}, nil)

	layer, err := newLayeredWorldState(m1, m2, m3, 1, common.HexToHash("0x11"))
	assert.Nil(t, err)
	assert.NotNil(t, layer)

	// Get from parent
	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x123"), false).
		Return(itypes.AccountValue{
			Nonce: 2,
		})
	acctp1 := layer.GetAccountValue(common.HexToAddress("0x123"), true)
	assert.Equal(t, uint64(2), acctp1.Nonce)

	// Get from cache
	acctp2 := layer.GetAccountValue(common.HexToAddress("0x123"), true)
	assert.Equal(t, uint64(2), acctp2.Nonce)

	// Get from current layer
	acctp3 := layer.GetAccountValue(common.HexToAddress(testAcctStr), true)
	assert.Equal(t, uint64(1), acctp3.Nonce)
}

func TestLayerGetCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	m3 := NewMockLayeredWorldState(ctrl)

	m1.
		EXPECT().
		GetLayerLog(uint64(1), common.HexToHash("0x11")).
		Return(itypes.LayerLog{
			CodePreimage: map[common.Hash][]byte{
				common.HexToHash("0x123"): {1, 2, 3},
			},
		}, nil)
	m2.
		EXPECT().
		Register(uint64(1), common.HexToHash("0x11"), gomock.Any())
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x11")).
		Return([]common.Hash{}, nil)

	layer, err := newLayeredWorldState(m1, m2, m3, 1, common.HexToHash("0x11"))
	assert.Nil(t, err)
	assert.NotNil(t, layer)

	// Get from parent
	m3.
		EXPECT().
		GetCodeByHash(common.HexToHash("0x124"), false).
		Return([]byte{1, 2, 4})
	code1 := layer.GetCodeByHash(common.HexToHash("0x124"), true)
	assert.Equal(t, []byte{1, 2, 4}, code1)

	// Get from cache
	code2 := layer.GetCodeByHash(common.HexToHash("0x124"), true)
	assert.Equal(t, []byte{1, 2, 4}, code2)

	// Get from current layer
	code3 := layer.GetCodeByHash(common.HexToHash("0x123"), true)
	assert.Equal(t, []byte{1, 2, 3}, code3)
}

func TestLayerGetStorageByVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	m3 := NewMockLayeredWorldState(ctrl)

	m1.
		EXPECT().
		GetLayerLog(uint64(1), common.HexToHash("0x11")).
		Return(itypes.LayerLog{
			UpdatedStorage: map[string]map[common.Hash]common.Hash{
				testAcctStr + "-4": {
					common.HexToHash("0x2345"): common.HexToHash("0x5432"),
				},
			},
		}, nil)
	m2.
		EXPECT().
		Register(uint64(1), common.HexToHash("0x11"), gomock.Any())
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x11")).
		Return([]common.Hash{}, nil)

	layer, err := newLayeredWorldState(m1, m2, m3, 1, common.HexToHash("0x11"))
	assert.Nil(t, err)
	assert.NotNil(t, layer)

	// Get from parent
	m3.
		EXPECT().
		GetStorageByVersion(common.HexToAddress("0x11"), uint64(5), common.HexToHash("0x1234"), false).
		Return(common.HexToHash("0x4321"))
	val1 := layer.GetStorageByVersion(common.HexToAddress("0x11"), 5, common.HexToHash("0x1234"), true)
	assert.Equal(t, common.HexToHash("0x4321"), val1)

	// Get from cache
	val2 := layer.GetStorageByVersion(common.HexToAddress("0x11"), 5, common.HexToHash("0x1234"), true)
	assert.Equal(t, common.HexToHash("0x4321"), val2)

	// Get from current layer
	val3 := layer.GetStorageByVersion(common.HexToAddress(testAcctStr), 4, common.HexToHash("0x2345"), true)
	assert.Equal(t, common.HexToHash("0x5432"), val3)
}

func TestLayerGetReadOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	m3 := NewMockLayeredWorldState(ctrl)

	m1.
		EXPECT().
		GetLayerLog(uint64(1), common.HexToHash("0x11")).
		Return(itypes.LayerLog{
			BlockNumber: 1,
		}, nil)
	m2.
		EXPECT().
		Register(uint64(1), common.HexToHash("0x11"), gomock.Any())
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x11")).
		Return([]common.Hash{}, nil)

	layer, err := newLayeredWorldState(m1, m2, m3, 1, common.HexToHash("0x11"))
	assert.Nil(t, err)
	assert.NotNil(t, layer)

	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)

	readOnly := layer.GetReadOnly()
	assert.NotNil(t, readOnly)
}

func TestLayerGetMutable(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	m3 := NewMockLayeredWorldState(ctrl)

	m1.
		EXPECT().
		GetLayerLog(uint64(1), common.HexToHash("0x11")).
		Return(itypes.LayerLog{
			BlockNumber: 1,
		}, nil)
	m2.
		EXPECT().
		Register(uint64(1), common.HexToHash("0x11"), gomock.Any())
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x11")).
		Return([]common.Hash{}, nil)

	layer, err := newLayeredWorldState(m1, m2, m3, 1, common.HexToHash("0x11"))
	assert.Nil(t, err)
	assert.NotNil(t, layer)

	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)

	mutable := layer.GetMutable()
	assert.NotNil(t, mutable)
}

func TestLayerDestructChil(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m1.
		EXPECT().
		GetLayerLog(uint64(1), common.HexToHash("0x11")).
		Return(itypes.LayerLog{
			BlockNumber: 1,
			RootHash:    common.HexToHash("0x11"),
		}, nil)
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x11")).
		Return([]common.Hash{common.HexToHash("0x12")}, nil)
	m1.
		EXPECT().
		GetLayerLog(uint64(2), common.HexToHash("0x12")).
		Return(itypes.LayerLog{
			BlockNumber: 2,
			RootHash:    common.HexToHash("0x12"),
		}, nil)
	m1.
		EXPECT().
		GetChildren(uint64(2), common.HexToHash("0x12")).
		Return([]common.Hash{}, nil)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	m2.
		EXPECT().
		Register(uint64(1), common.HexToHash("0x11"), gomock.Any())
	m2.
		EXPECT().
		Register(uint64(2), common.HexToHash("0x12"), gomock.Any())
	m3 := NewMockLayeredWorldState(ctrl)

	layer, err := newLayeredWorldState(m1, m2, m3, 1, common.HexToHash("0x11"))
	assert.Nil(t, err)
	assert.NotNil(t, layer)

	// Delete not existed child has on effect
	layer.DestructChild(common.HexToHash("0x13"))

	// Delete existed child generate calls
	txn := statestore.NewMockTransaction(ctrl)
	txn.
		EXPECT().
		DeleteChildren(uint64(2), common.HexToHash("0x12")).
		Return(nil)
	txn.
		EXPECT().
		DeleteLayerLog(uint64(2), common.HexToHash("0x12")).
		Return(nil)
	txn.
		EXPECT().
		Commit().
		Return(nil)
	txn.
		EXPECT().
		Discard().
		AnyTimes()
	m2.
		EXPECT().
		Deregister(uint64(2), common.HexToHash("0x12"))
	txn.
		EXPECT().
		PutChildren(uint64(1), common.HexToHash("0x11"), []common.Hash{}).
		Return(nil)
	txn.
		EXPECT().
		Commit().
		Return(nil)
	m1.
		EXPECT().
		NewTransaction().
		Return(txn, nil).
		AnyTimes()
	layer.DestructChild(common.HexToHash("0x12"))
}

func TestLayerAddChild(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m1.
		EXPECT().
		GetLayerLog(uint64(1), common.HexToHash("0x11")).
		Return(itypes.LayerLog{
			BlockNumber: 1,
			RootHash:    common.HexToHash("0x11"),
		}, nil)
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x11")).
		Return([]common.Hash{common.HexToHash("0x12")}, nil)
	m1.
		EXPECT().
		GetLayerLog(uint64(2), common.HexToHash("0x12")).
		Return(itypes.LayerLog{
			BlockNumber: 2,
			RootHash:    common.HexToHash("0x12"),
		}, nil)
	m1.
		EXPECT().
		GetChildren(uint64(2), common.HexToHash("0x12")).
		Return([]common.Hash{}, nil)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	m2.
		EXPECT().
		Register(uint64(1), common.HexToHash("0x11"), gomock.Any())
	m2.
		EXPECT().
		Register(uint64(2), common.HexToHash("0x12"), gomock.Any())
	m3 := NewMockLayeredWorldState(ctrl)
	layer, err := newLayeredWorldState(m1, m2, m3, 1, common.HexToHash("0x11"))
	assert.Nil(t, err)
	assert.NotNil(t, layer)

	m4 := NewMockLayeredWorldState(ctrl)
	m4.
		EXPECT().
		GetRootHash().
		Return(common.HexToHash("0x13"))
	txn := statestore.NewMockTransaction(ctrl)
	txn.
		EXPECT().
		PutChildren(uint64(1), common.HexToHash("0x11"), []common.Hash{common.HexToHash("0x12"), common.HexToHash("0x13")}).
		Return(nil)
	txn.
		EXPECT().
		Commit().
		Return(nil)
	txn.
		EXPECT().
		Discard()
	m1.
		EXPECT().
		NewTransaction().
		Return(txn, nil)
	layer.AddChild(m4)
}
