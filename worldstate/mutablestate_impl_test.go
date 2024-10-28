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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/teler/statestore"
	itypes "github.com/wcgcyx/teler/types"
	"go.uber.org/mock/gomock"
)

func TestNewMutableState(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)
}

func TestMutableStateExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	// Test account that does not exist
	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Version: 0,
			Balance: uint256.NewInt(0),
		})
	assert.False(t, mutable.Exist(common.HexToAddress("0x11")))

	// Test account that does exist
	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x12"), true).
		Return(itypes.AccountValue{
			Version: 1,
			Balance: uint256.NewInt(1),
		})
	assert.True(t, mutable.Exist(common.HexToAddress("0x12")))
}

func TestMutableStateEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	// Test account that is empty
	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        0,
			Version:      0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
		})
	assert.True(t, mutable.Empty(common.HexToAddress("0x11")))
}

func TestMutableGetBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        0,
			Version:      1,
			Balance:      uint256.NewInt(100),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
		})

	assert.Equal(t, uint256.NewInt(100), mutable.GetBalance(common.HexToAddress("0x11")))
}

func TestMutableGetNonce(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        2,
			Version:      1,
			Balance:      uint256.NewInt(100),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
		})

	assert.Equal(t, uint64(2), mutable.GetNonce(common.HexToAddress("0x11")))
}

func TestMutableGetCodeHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        2,
			Version:      1,
			Balance:      uint256.NewInt(100),
			CodeHash:     common.HexToHash("0x123"),
			DirtyStorage: false,
		})

	assert.Equal(t, common.HexToHash("0x123"), mutable.GetCodeHash(common.HexToAddress("0x11")))
}

func TestMutableGetCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        2,
			Version:      1,
			Balance:      uint256.NewInt(100),
			CodeHash:     common.HexToHash("0x123"),
			DirtyStorage: false,
		})
	m3.
		EXPECT().
		GetCodeByHash(common.HexToHash("0x123"), true).
		Return([]byte{1, 2, 3})

	assert.Equal(t, []byte{1, 2, 3}, mutable.GetCode(common.HexToAddress("0x11")))
}

func TestMutableGetCodeSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        2,
			Version:      1,
			Balance:      uint256.NewInt(100),
			CodeHash:     common.HexToHash("0x123"),
			DirtyStorage: false,
		})
	m3.
		EXPECT().
		GetCodeByHash(common.HexToHash("0x123"), true).
		Return([]byte{1, 2, 3})

	assert.Equal(t, 3, mutable.GetCodeSize(common.HexToAddress("0x11")))
}

func TestMutableGetStorageRoot(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        2,
			Version:      1,
			Balance:      uint256.NewInt(100),
			CodeHash:     common.HexToHash("0x123"),
			DirtyStorage: false,
		})
	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x12"), true).
		Return(itypes.AccountValue{
			Nonce:        2,
			Version:      1,
			Balance:      uint256.NewInt(100),
			CodeHash:     common.HexToHash("0x123"),
			DirtyStorage: true,
		})

	assert.Equal(t, types.EmptyRootHash, mutable.GetStorageRoot(common.HexToAddress("0x11")))
	assert.Equal(t, common.HexToHash("1"), mutable.GetStorageRoot(common.HexToAddress("0x12")))
}

func TestMutableGetState(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        2,
			Version:      1,
			Balance:      uint256.NewInt(100),
			CodeHash:     common.HexToHash("0x123"),
			DirtyStorage: true,
		})
	m3.
		EXPECT().
		GetStorageByVersion(common.HexToAddress("0x11"), uint64(1), common.HexToHash("0x123"), true).
		Return(common.HexToHash("0x321"))

	assert.Equal(t, common.HexToHash("0x321"), mutable.GetState(common.HexToAddress("0x11"), common.HexToHash("0x123")))
}

func TestMutableSetTxContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	assert.Equal(t, -1, mutable.TxIndex())
	mutable.SetTxContext(common.HexToHash("0x123"), 1)
	assert.Equal(t, 1, mutable.TxIndex())
}

func TestMutablePrepare(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	mutable.Prepare(
		params.AllEthashProtocolChanges.Rules(big.NewInt(100), false, 1),
		common.HexToAddress("0x123"),
		common.HexToAddress("0x124"),
		nil,
		[]common.Address{},
		types.AccessList{})
}

func TestCreateAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        0,
			Version:      0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
		}).AnyTimes()

	assert.False(t, mutable.Exist(common.HexToAddress("0x11")))
	mutable.CreateAccount(common.HexToAddress("0x11"))
	assert.True(t, mutable.Exist(common.HexToAddress("0x11")))
}

func TestCreateContract(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        0,
			Version:      0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
		}).AnyTimes()

	assert.False(t, mutable.Exist(common.HexToAddress("0x11")))
	mutable.CreateContract(common.HexToAddress("0x11"))
	// Create contract only sets contract tag
	assert.False(t, mutable.Exist(common.HexToAddress("0x11")))
}

func TestAccountBalanceChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        0,
			Version:      0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
		}).AnyTimes()

	assert.False(t, mutable.Exist(common.HexToAddress("0x11")))
	mutable.CreateAccount(common.HexToAddress("0x11"))
	assert.True(t, mutable.Exist(common.HexToAddress("0x11")))

	assert.Equal(t, uint256.NewInt(0), mutable.GetBalance(common.HexToAddress("0x11")))
	mutable.AddBalance(common.HexToAddress("0x11"), uint256.NewInt(10), tracing.BalanceChangeTransfer)
	assert.Equal(t, uint256.NewInt(10), mutable.GetBalance(common.HexToAddress("0x11")))
	mutable.SubBalance(common.HexToAddress("0x11"), uint256.NewInt(5), tracing.BalanceChangeTransfer)
	assert.Equal(t, uint256.NewInt(5), mutable.GetBalance(common.HexToAddress("0x11")))
	mutable.SetBalance(common.HexToAddress("0x11"), uint256.NewInt(50), tracing.BalanceChangeTransfer)
	assert.Equal(t, uint256.NewInt(50), mutable.GetBalance(common.HexToAddress("0x11")))
}

func TestAccountSetNonce(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        0,
			Version:      0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
		}).AnyTimes()

	mutable.CreateAccount(common.HexToAddress("0x11"))
	assert.Equal(t, uint64(0), mutable.GetNonce(common.HexToAddress("0x11")))
	mutable.SetNonce(common.HexToAddress("0x11"), 1)
	assert.Equal(t, uint64(1), mutable.GetNonce(common.HexToAddress("0x11")))
}

func TestAccountSetCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        0,
			Version:      0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
		}).AnyTimes()

	mutable.CreateAccount(common.HexToAddress("0x11"))
	assert.Equal(t, []byte{}, mutable.GetCode(common.HexToAddress("0x11")))
	mutable.SetCode(common.HexToAddress("0x11"), []byte{1, 2, 3})
	assert.Equal(t, []byte{1, 2, 3}, mutable.GetCode(common.HexToAddress("0x11")))
}

func TestAccountSetState(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        0,
			Version:      0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
		}).AnyTimes()

	mutable.CreateAccount(common.HexToAddress("0x11"))
	assert.Equal(t, common.Hash{}, mutable.GetState(common.HexToAddress("0x11"), common.HexToHash("0x123")))
	mutable.SetState(common.HexToAddress("0x11"), common.HexToHash("0x123"), common.HexToHash("0x321"))
	assert.Equal(t, common.HexToHash("0x321"), mutable.GetState(common.HexToAddress("0x11"), common.HexToHash("0x123")))
}

func TestGetCommittedState(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        0,
			Version:      0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
		}).AnyTimes()

	mutable.CreateAccount(common.HexToAddress("0x11"))
	mutable.SetState(common.HexToAddress("0x11"), common.HexToHash("0x123"), common.HexToHash("0x321"))
	assert.Equal(t, common.HexToHash("0x321"), mutable.GetState(common.HexToAddress("0x11"), common.HexToHash("0x123")))
	assert.Equal(t, common.Hash{}, mutable.GetCommittedState(common.HexToAddress("0x11"), common.HexToHash("0x123")))
}

func TestSetTransientState(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	assert.Equal(t, common.Hash{}, mutable.GetTransientState(common.HexToAddress("0x11"), common.HexToHash("0x123")))
	mutable.SetTransientState(common.HexToAddress("0x11"), common.HexToHash("0x123"), common.HexToHash("0x321"))
	assert.Equal(t, common.HexToHash("0x321"), mutable.GetTransientState(common.HexToAddress("0x11"), common.HexToHash("0x123")))
}

func TestGasRefund(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	assert.Equal(t, uint64(0), mutable.GetRefund())
	mutable.AddRefund(2)
	assert.Equal(t, uint64(2), mutable.GetRefund())
	mutable.SubRefund(1)
	assert.Equal(t, uint64(1), mutable.GetRefund())
}

func TestSelfDestruct(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        1,
			Version:      1,
			Balance:      uint256.NewInt(11),
			CodeHash:     common.HexToHash("0x123"),
			DirtyStorage: true,
		}).AnyTimes()

	assert.False(t, mutable.HasSelfDestructed(common.HexToAddress("0x11")))
	mutable.SelfDestruct(common.HexToAddress("0x11"))
	assert.True(t, mutable.HasSelfDestructed(common.HexToAddress("0x11")))
}

func TestSelfDestruct6780(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        1,
			Version:      1,
			Balance:      uint256.NewInt(11),
			CodeHash:     common.HexToHash("0x123"),
			DirtyStorage: true,
		}).AnyTimes()

	assert.False(t, mutable.HasSelfDestructed(common.HexToAddress("0x11")))
	mutable.Selfdestruct6780(common.HexToAddress("0x11"))
	assert.False(t, mutable.HasSelfDestructed(common.HexToAddress("0x11")))

	mutable.CreateContract(common.HexToAddress("0x11"))
	mutable.Selfdestruct6780(common.HexToAddress("0x11"))
	assert.True(t, mutable.HasSelfDestructed(common.HexToAddress("0x11")))
}

func TestAddressInAccessList(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	assert.False(t, mutable.AddressInAccessList(common.HexToAddress("0x11")))
	mutable.AddAddressToAccessList(common.HexToAddress("0x11"))
	assert.True(t, mutable.AddressInAccessList(common.HexToAddress("0x11")))
}

func TestSlotInAccessList(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	addressOk, slotOk := mutable.SlotInAccessList(common.HexToAddress("0x11"), common.HexToHash("0x123"))
	assert.False(t, addressOk)
	assert.False(t, slotOk)

	mutable.AddSlotToAccessList(common.HexToAddress("0x11"), common.HexToHash("0x123"))
	mutable.AddSlotToAccessList(common.HexToAddress("0x11"), common.HexToHash("0x124"))

	addressOk, slotOk = mutable.SlotInAccessList(common.HexToAddress("0x11"), common.HexToHash("0x125"))
	assert.True(t, addressOk)
	assert.False(t, slotOk)

	addressOk, slotOk = mutable.SlotInAccessList(common.HexToAddress("0x11"), common.HexToHash("0x123"))
	assert.True(t, addressOk)
	assert.True(t, slotOk)
}

func TestPointCache(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("should panic")
		}
	}()

	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	mutable.PointCache()
}

func TestMutableGetLogs(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	assert.Equal(t, 0, len(mutable.GetLogs(common.HexToHash("0x123"), 123, common.HexToHash("0x321"))))
	mutable.SetTxContext(common.HexToHash("0x123"), 1)
	mutable.AddLog(&types.Log{})
	assert.Equal(t, 0, len(mutable.GetLogs(common.HexToHash("0x124"), 123, common.HexToHash("0x321"))))
	assert.Equal(t, 1, len(mutable.GetLogs(common.HexToHash("0x123"), 123, common.HexToHash("0x321"))))
}

func TestSnapsho(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        0,
			Version:      0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
		}).AnyTimes()

	assert.False(t, mutable.Exist(common.HexToAddress("0x11")))
	snapId := mutable.Snapshot()
	mutable.CreateAccount(common.HexToAddress("0x11"))
	assert.True(t, mutable.Exist(common.HexToAddress("0x11")))
	mutable.RevertToSnapshot(snapId)
	assert.False(t, mutable.Exist(common.HexToAddress("0x11")))
}

func TestFinalise(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        0,
			Version:      0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
		}).AnyTimes()

	mutable.CreateAccount(common.HexToAddress("0x11"))

	mutable.Finalise(true)
}

func TestIntermediateRoot(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        0,
			Version:      0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
		}).AnyTimes()

	mutable.CreateAccount(common.HexToAddress("0x11"))

	assert.Equal(t, types.EmptyRootHash, mutable.IntermediateRoot(true))
}

func TestCommit(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress("0x11"), true).
		Return(itypes.AccountValue{
			Nonce:        0,
			Version:      0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
		}).AnyTimes()

	mutable.CreateAccount(common.HexToAddress("0x11"))

	m2.
		EXPECT().
		Has(uint64(1), common.HexToHash("0x123")).
		Return(false)
	txn := statestore.NewMockTransaction(ctrl)
	txn.
		EXPECT().
		PutChildren(uint64(1), common.HexToHash("0x123"), []common.Hash{}).
		Return(nil)
	txn.
		EXPECT().
		PutLayerLog(gomock.Any()).
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
	m1.
		EXPECT().
		GetLayerLog(uint64(1), common.HexToHash("0x123")).
		Return(itypes.LayerLog{}, nil)
	m2.
		EXPECT().
		Register(uint64(1), common.HexToHash("0x123"), gomock.Any())
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x123")).
		Return([]common.Hash{}, nil)
	m3.
		EXPECT().
		AddChild(gomock.Any())
	m3.
		EXPECT().
		HandlePrune(gomock.Any())
	mutable.Commit(1, common.HexToHash("0x123"), true)
}

func TestMutableCopy(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := statestore.NewMockStateStore(ctrl)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	genesis := core.DefaultGenesisBlock()
	m2.
		EXPECT().
		GetChainConfig().
		Return(genesis.Config)
	m3 := NewMockLayeredWorldState(ctrl)

	mutable := newMutableWorldState(m1, m2, m3, 1, nil)
	assert.NotNil(t, mutable)

	mutable2 := mutable.Copy()
	assert.NotNil(t, mutable2)
}
