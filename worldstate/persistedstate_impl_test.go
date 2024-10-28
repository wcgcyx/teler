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
	"fmt"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/teler/statestore"
	itypes "github.com/wcgcyx/teler/types"
	"go.uber.org/mock/gomock"
)

func TestNewPersistedState(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m1.
		EXPECT().
		GetPersistedHeight().
		Return(uint64(1), common.HexToHash("0x11"), nil)
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x11")).
		Return([]common.Hash{common.HexToHash("0x12")}, nil)
	m1.
		EXPECT().
		GetLayerLog(uint64(2), common.HexToHash("0x12")).
		Return(itypes.LayerLog{}, nil)
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
	layer, err := newPersistedWorldState(m1, m2)
	assert.Nil(t, err)
	assert.NotNil(t, layer)
}

func TestGetAccountValue(t *testing.T) {
	expectedValue := itypes.AccountValue{
		Nonce:        1,
		Balance:      uint256.NewInt(2),
		CodeHash:     common.HexToHash("0x1"),
		DirtyStorage: true,
		Version:      1,
	}

	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m1.
		EXPECT().
		GetPersistedHeight().
		Return(uint64(1), common.HexToHash("0x11"), nil)
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x11")).
		Return([]common.Hash{}, nil)
	m1.
		EXPECT().
		GetAccountValue(common.HexToAddress(testAcctStr)).
		Return(itypes.AccountValue{}, fmt.Errorf("fail"))
	m1.
		EXPECT().
		GetAccountValue(common.HexToAddress(testAcctStr)).
		Return(expectedValue, nil)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	m2.
		EXPECT().
		Register(uint64(1), common.HexToHash("0x11"), gomock.Any())
	layer, err := newPersistedWorldState(m1, m2)
	assert.Nil(t, err)
	assert.NotNil(t, layer)

	// Get with error
	acct := layer.GetAccountValue(common.HexToAddress(testAcctStr), true)
	assert.Equal(t, itypes.AccountValue{}, acct)

	// Get without error
	acct = layer.GetAccountValue(common.HexToAddress(testAcctStr), true)
	assert.Equal(t, expectedValue, acct)

	// Get with cache
	acct = layer.GetAccountValue(common.HexToAddress(testAcctStr), true)
	assert.Equal(t, expectedValue, acct)
}

func TestGetCodeByHash(t *testing.T) {
	expectedCode := []byte{1, 2, 3}

	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	m1.
		EXPECT().
		GetPersistedHeight().
		Return(uint64(1), common.HexToHash("0x11"), nil)
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x11")).
		Return([]common.Hash{}, nil)
	m1.
		EXPECT().
		GetCodeByHash(common.HexToHash("0x123")).
		Return(nil, fmt.Errorf("fail"))
	m1.
		EXPECT().
		GetCodeByHash(common.HexToHash("0x123")).
		Return(expectedCode, nil)
	m2.
		EXPECT().
		Register(uint64(1), common.HexToHash("0x11"), gomock.Any())

	layer, err := newPersistedWorldState(m1, m2)
	assert.Nil(t, err)
	assert.NotNil(t, layer)

	// Get with error
	code := layer.GetCodeByHash(common.HexToHash("0x123"), true)
	assert.Equal(t, []byte{}, code)

	// Get without error
	code = layer.GetCodeByHash(common.HexToHash("0x123"), true)
	assert.Equal(t, expectedCode, code)

	// Get with cache
	code = layer.GetCodeByHash(common.HexToHash("0x123"), true)
	assert.Equal(t, expectedCode, code)
}

func TestGetStorageByVersion(t *testing.T) {
	expectedStorage := common.HexToHash("0x123")

	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	m1.
		EXPECT().
		GetPersistedHeight().
		Return(uint64(1), common.HexToHash("0x11"), nil)
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x11")).
		Return([]common.Hash{}, nil)
	m1.
		EXPECT().
		GetStorageByVersion(common.HexToAddress(testAcctStr), uint64(1), common.HexToHash("0x1")).
		Return(common.Hash{}, fmt.Errorf("fail"))
	m1.
		EXPECT().
		GetStorageByVersion(common.HexToAddress(testAcctStr), uint64(1), common.HexToHash("0x1")).
		Return(expectedStorage, nil)
	m2.
		EXPECT().
		Register(uint64(1), common.HexToHash("0x11"), gomock.Any())

	layer, err := newPersistedWorldState(m1, m2)
	assert.Nil(t, err)
	assert.NotNil(t, layer)

	// Get with error
	val := layer.GetStorageByVersion(common.HexToAddress(testAcctStr), 1, common.HexToHash("0x1"), true)
	assert.Equal(t, common.Hash{}, val)

	// Get without error
	val = layer.GetStorageByVersion(common.HexToAddress(testAcctStr), 1, common.HexToHash("0x1"), true)
	assert.Equal(t, expectedStorage, val)

	// Get with cache
	val = layer.GetStorageByVersion(common.HexToAddress(testAcctStr), 1, common.HexToHash("0x1"), true)
	assert.Equal(t, expectedStorage, val)
}

func TestGetReadOnly(t *testing.T) {
	genesis := core.DefaultGenesisBlock()
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	m1.
		EXPECT().
		GetPersistedHeight().
		Return(uint64(1), common.HexToHash("0x11"), nil)
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x11")).
		Return([]common.Hash{}, nil)
	m2.
		EXPECT().
		Register(uint64(1), common.HexToHash("0x11"), gomock.Any())
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)

	layer, err := newPersistedWorldState(m1, m2)
	assert.Nil(t, err)
	assert.NotNil(t, layer)
	assert.NotNil(t, layer.GetReadOnly())
}

func TestHandlePrune(t *testing.T) {
	genesis := core.DefaultGenesisBlock()
	// Create a chain of 300 states
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m1.
		EXPECT().
		GetPersistedHeight().
		Return(uint64(1), common.HexToHash("0x11"), nil)
	for i := uint64(1); i < 300; i++ {
		currentHash := common.HexToHash(strconv.FormatUint(16+i, 16))
		childHash := common.HexToHash(strconv.FormatUint(17+i, 16))
		m1.
			EXPECT().
			GetChildren(uint64(i), currentHash).
			Return([]common.Hash{childHash}, nil)
		m1.
			EXPECT().
			GetLayerLog(i+1, childHash).
			Return(itypes.LayerLog{
				BlockNumber: i + 1,
				RootHash:    childHash,
			}, nil)
	}
	m1.
		EXPECT().
		GetChildren(uint64(300), common.HexToHash(strconv.FormatUint(316, 16))).
		Return([]common.Hash{}, nil)

	txn := statestore.NewMockTransaction(ctrl)
	for i := uint64(1); i <= 128; i++ {
		txn.
			EXPECT().
			DeleteChildren(i, common.HexToHash(strconv.FormatUint(16+i, 16))).
			Return(nil)
		txn.
			EXPECT().
			DeleteLayerLog(i+1, common.HexToHash(strconv.FormatUint(16+i+1, 16))).
			Return(nil)
		txn.
			EXPECT().
			PersistLayerLog(gomock.Any()).
			Return(nil)
		txn.
			EXPECT().
			Commit().
			Return(nil)
	}
	txn.
		EXPECT().
		Discard().
		AnyTimes()
	m1.
		EXPECT().
		NewTransaction().
		Return(txn, nil).
		AnyTimes()

	sa, err := NewLayeredWorldStateArchiveImpl(Opts{
		MaxLayerToRetain: 256,
		PruningFrequency: 128,
	}, genesis.Config, m1)
	assert.Nil(t, err)
	assert.NotNil(t, sa)

	layer, err := sa.GetLayared(300, common.HexToHash(strconv.FormatUint(316, 16)))
	assert.Nil(t, err)

	layer.HandlePrune([]common.Hash{})
}

func TestDestructChild(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m1.
		EXPECT().
		GetPersistedHeight().
		Return(uint64(1), common.HexToHash("0x11"), nil)
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
	layer, err := newPersistedWorldState(m1, m2)
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

func TestAddChild(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m1.
		EXPECT().
		GetPersistedHeight().
		Return(uint64(1), common.HexToHash("0x11"), nil)
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
	layer, err := newPersistedWorldState(m1, m2)
	assert.Nil(t, err)
	assert.NotNil(t, layer)

	m3 := NewMockLayeredWorldState(ctrl)
	m3.
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
	layer.AddChild(m3)
}
