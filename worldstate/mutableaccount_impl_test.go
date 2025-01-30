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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	itypes "github.com/wcgcyx/teler/types"
	"go.uber.org/mock/gomock"
)

const (
	testAcctStr = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
)

func TestNewAccount(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(2),
			CodeHash:     common.HexToHash("0x3"),
			DirtyStorage: true,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))

	assert.Equal(t, uint64(1), acct.Nonce())
	assert.Equal(t, uint256.NewInt(2), acct.Balance())
	assert.Equal(t, common.HexToHash("0x3"), acct.CodeHash())
	assert.Equal(t, true, acct.DirtyStorage())
}

func TestAccountEmptiness(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.False(t, acct.Empty())

	acct = newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(1),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.False(t, acct.Empty())

	acct = newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     common.HexToHash("1"),
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.False(t, acct.Empty())

	acct = newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.True(t, acct.Empty())
}

func TestAccountExistence(t *testing.T) {
	// pre eip 158 check
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      0,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.False(t, acct.Exists(false))

	acct = newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.True(t, acct.Exists(false))

	// post eip 158 check
	acct = newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.False(t, acct.Exists(true))

	acct = newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.True(t, acct.Exists(true))
}

func TestGetCodeWithEmptyCodeHash(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.Equal(t, []byte{}, acct.Code())
}

func TestGetCodeFromKnownCode(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(0),
			CodeHash:     common.HexToHash("0x1"),
			DirtyStorage: false,
			Version:      1,
		},
		[]byte{1, 2, 3},
		make(map[common.Hash]common.Hash))
	assert.Equal(t, []byte{1, 2, 3}, acct.Code())
}

func TestGetCodeFromParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockLayeredWorldState(ctrl)
	m.
		EXPECT().
		GetCodeByHash(common.HexToHash("0x1"), true).
		Return([]byte{1, 2, 3})

	acct := newMutableAccountImpl(
		m,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(0),
			CodeHash:     common.HexToHash("0x1"),
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.Equal(t, []byte{1, 2, 3}, acct.Code())
}

func TestGetStorageWithCleanStorage(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.Equal(t, common.Hash{}, acct.GetState(common.HexToHash("0x1")))
	assert.Equal(t, common.Hash{}, acct.GetState(common.HexToHash("0x2")))
}

func TestGetStorageFromKnownStorage(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: true,
			Version:      1,
		},
		nil,
		map[common.Hash]common.Hash{
			common.HexToHash("0x1"): common.HexToHash("0x11"),
			common.HexToHash("0x2"): common.HexToHash("0x22"),
		},
	)
	assert.Equal(t, common.HexToHash("0x11"), acct.GetState(common.HexToHash("0x1")))
	assert.Equal(t, common.HexToHash("0x22"), acct.GetState(common.HexToHash("0x2")))
}

func TestGetStorageFromParentForUpdatingAcct(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockLayeredWorldState(ctrl)
	m.
		EXPECT().
		GetStorageByVersion(common.HexToAddress(testAcctStr), uint64(1), common.HexToHash("0x1"), true).
		Return(common.HexToHash("0x11"))
	m.
		EXPECT().
		GetStorageByVersion(common.HexToAddress(testAcctStr), uint64(1), common.HexToHash("0x2"), true).
		Return(common.HexToHash("0x22"))

	acct := newMutableAccountImpl(
		m,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(0),
			CodeHash:     common.HexToHash("0x1"),
			DirtyStorage: true,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.Equal(t, common.HexToHash("0x11"), acct.GetState(common.HexToHash("0x1")))
	assert.Equal(t, common.HexToHash("0x22"), acct.GetState(common.HexToHash("0x2")))
}
func TestGetStorageFromParentForDeletingAcct(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockLayeredWorldState(ctrl)
	m.
		EXPECT().
		GetStorageByVersion(common.HexToAddress(testAcctStr), uint64(1), common.HexToHash("0x1"), true).
		Return(common.HexToHash("0x11"))
	m.
		EXPECT().
		GetStorageByVersion(common.HexToAddress(testAcctStr), uint64(1), common.HexToHash("0x2"), true).
		Return(common.HexToHash("0x22"))

	acct := newMutableAccountImpl(
		m,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(0),
			CodeHash:     common.HexToHash("0x1"),
			DirtyStorage: true,
			Version:      2,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.Equal(t, common.HexToHash("0x11"), acct.GetState(common.HexToHash("0x1")))
	assert.Equal(t, common.HexToHash("0x22"), acct.GetState(common.HexToHash("0x2")))
}

func TestCreateToOverwriteDeletingAccount(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("should panic")
		}
	}()

	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      2,
		},
		nil,
		make(map[common.Hash]common.Hash))
	acct.Create()
}

func TestCreateToOverwriteNonEmptyAccount(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("should panic")
		}
	}()

	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	acct.Create()
}

func TestCreateToOverwritEmptyAccount(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	acct.Create()
}

func TestCreateFromNonExistedAccount(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      0,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.Equal(t, uint64(0), acct.Version())
	revert := acct.Create()
	assert.Equal(t, uint64(1), acct.Version())
	revert()
	assert.Equal(t, uint64(0), acct.Version())
}

func TestSetNonceOnNonExistedAccount(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      0,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.Equal(t, uint64(0), acct.Version())
	assert.Equal(t, uint64(0), acct.Nonce())
	revert := acct.SetNonce(1)
	assert.Equal(t, uint64(1), acct.Version())
	assert.Equal(t, uint64(1), acct.Nonce())
	revert()
	assert.Equal(t, uint64(0), acct.Version())
	assert.Equal(t, uint64(0), acct.Nonce())
}

func TestSetNonceOnDeletingAccount(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      2,
		},
		nil,
		make(map[common.Hash]common.Hash))

	acct.SetNonce(1)
	assert.Equal(t, uint64(1), acct.Nonce())
}

func TestSetNonceOnUpdatingAccount(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.Equal(t, uint64(0), acct.Nonce())
	revert := acct.SetNonce(0)
	assert.Equal(t, uint64(0), acct.Nonce())
	revert()
	assert.Equal(t, uint64(0), acct.Nonce())
	revert = acct.SetNonce(1)
	assert.Equal(t, uint64(1), acct.Nonce())
	revert()
	assert.Equal(t, uint64(0), acct.Nonce())
}

func TestSetBalanceOnNonExistedAccount(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      0,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.Equal(t, uint64(0), acct.Version())
	assert.Equal(t, uint256.NewInt(0), acct.Balance())
	revert := acct.SetBalance(uint256.NewInt(1))
	assert.Equal(t, uint64(1), acct.Version())
	assert.Equal(t, uint256.NewInt(1), acct.Balance())
	revert()
	assert.Equal(t, uint64(0), acct.Version())
	assert.Equal(t, uint256.NewInt(0), acct.Balance())
}

func TestSetBalanceOnDeletingAccount(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      2,
		},
		nil,
		make(map[common.Hash]common.Hash))

	assert.Equal(t, uint64(2), acct.Version())
	assert.Equal(t, uint256.NewInt(0), acct.Balance())
	revert := acct.SetBalance(uint256.NewInt(1))
	assert.Equal(t, uint64(2), acct.Version())
	assert.Equal(t, uint256.NewInt(1), acct.Balance())
	revert()
	assert.Equal(t, uint64(2), acct.Version())
	assert.Equal(t, uint256.NewInt(0), acct.Balance())
}

func TestSetBalanceOnUpdatingAccount(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.Equal(t, uint256.NewInt(0), acct.Balance())
	revert := acct.SetBalance(uint256.NewInt(0))
	assert.Equal(t, uint256.NewInt(0), acct.Balance())
	revert()
	assert.Equal(t, uint256.NewInt(0), acct.Balance())
	revert = acct.SetBalance(uint256.NewInt(1))
	assert.Equal(t, uint256.NewInt(1), acct.Balance())
	revert()
	assert.Equal(t, uint256.NewInt(0), acct.Balance())
}

func TestSetCodeOnNonExistedAccount(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      0,
		},
		nil,
		make(map[common.Hash]common.Hash))
	revert := acct.SetCode([]byte{1})
	assert.Equal(t, uint64(1), acct.Version())
	assert.Equal(t, []byte{1}, acct.Code())
	assert.Equal(t, crypto.Keccak256Hash([]byte{1}), acct.CodeHash())
	revert()
	assert.Equal(t, uint64(0), acct.Version())
	assert.Equal(t, []byte{}, acct.Code())
	assert.Equal(t, types.EmptyCodeHash, acct.CodeHash())
}

func TestSetZeroCodeOnDeletingAccount(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     common.HexToHash("0x1"),
			DirtyStorage: false,
			Version:      2,
		},
		nil,
		make(map[common.Hash]common.Hash))
	revert := acct.SetCode([]byte{})
	assert.Equal(t, uint64(2), acct.Version())
	assert.Equal(t, []byte{}, acct.Code())
	assert.Equal(t, types.EmptyCodeHash, acct.CodeHash())
	revert()
	assert.Equal(t, uint64(2), acct.Version())
	assert.Equal(t, common.HexToHash("0x1"), acct.CodeHash())
}

func TestSetNonZeroCodeOnDeletingAccount(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("should panic")
		}
	}()

	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     common.HexToHash("0x1"),
			DirtyStorage: false,
			Version:      2,
		},
		nil,
		make(map[common.Hash]common.Hash))
	acct.SetCode([]byte{1})
}

func TestSetCodeOnUpdatingAccount(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	revert := acct.SetCode([]byte{})
	assert.Equal(t, uint64(1), acct.Version())
	assert.Equal(t, []byte{}, acct.Code())
	assert.Equal(t, types.EmptyCodeHash, acct.CodeHash())
	revert()
	revert = acct.SetCode([]byte{1})
	assert.Equal(t, uint64(1), acct.Version())
	assert.Equal(t, []byte{1}, acct.Code())
	assert.Equal(t, crypto.Keccak256Hash([]byte{1}), acct.CodeHash())
	revert()
	assert.Equal(t, uint64(1), acct.Version())
	assert.Equal(t, []byte{}, acct.Code())
	assert.Equal(t, types.EmptyCodeHash, acct.CodeHash())
}

func TestSetDifferentCodeOnUpdatingAccount(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("should panic")
		}
	}()

	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     common.HexToHash("0x1"),
			DirtyStorage: false,
			Version:      1,
		},
		nil,
		make(map[common.Hash]common.Hash))
	acct.SetCode([]byte{1})
}

func TestSetStateOnNonExistingAccount(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      0,
		},
		nil,
		make(map[common.Hash]common.Hash))
	assert.Equal(t, uint64(0), acct.Version())
	assert.Equal(t, common.Hash{}, acct.GetState(common.HexToHash("0x1")))
	revert := acct.SetState(common.HexToHash("0x1"), common.HexToHash("0x11"))
	assert.Equal(t, uint64(1), acct.Version())
	assert.Equal(t, common.HexToHash("0x11"), acct.GetState(common.HexToHash("0x1")))
	revert()
	assert.Equal(t, uint64(0), acct.Version())
	assert.Equal(t, common.Hash{}, acct.GetState(common.HexToHash("0x1")))
}

func TestSetStateOnExistingAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockLayeredWorldState(ctrl)
	m.
		EXPECT().
		GetStorageByVersion(common.HexToAddress(testAcctStr), uint64(1), common.HexToHash("0x2"), true).
		Return(common.Hash{})

	acct := newMutableAccountImpl(
		m,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: true,
			Version:      1,
		},
		nil,
		map[common.Hash]common.Hash{
			common.HexToHash("0x1"): common.HexToHash("0x11"),
		})
	assert.Equal(t, common.HexToHash("0x11"), acct.GetState(common.HexToHash("0x1")))
	revert := acct.SetState(common.HexToHash("0x1"), common.HexToHash("0x11"))
	assert.Equal(t, common.HexToHash("0x11"), acct.GetState(common.HexToHash("0x1")))
	revert()
	assert.Equal(t, common.HexToHash("0x11"), acct.GetState(common.HexToHash("0x1")))

	revert = acct.SetState(common.HexToHash("0x1"), common.HexToHash("0x12"))
	assert.Equal(t, common.HexToHash("0x12"), acct.GetState(common.HexToHash("0x1")))
	revert()
	assert.Equal(t, common.HexToHash("0x11"), acct.GetState(common.HexToHash("0x1")))

	revert = acct.SetState(common.HexToHash("0x2"), common.HexToHash("0x22"))
	assert.Equal(t, common.HexToHash("0x22"), acct.GetState(common.HexToHash("0x2")))
	revert()
	assert.Equal(t, common.Hash{}, acct.GetState(common.HexToHash("0x2")))
}

func TestSelfDestructNonExistedAccount(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("should panic")
		}
	}()

	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      0,
		},
		nil,
		make(map[common.Hash]common.Hash))

	acct.SelfDestruct()
}

func TestSelfDestructAccount(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: true,
			Version:      1,
		},
		nil,
		map[common.Hash]common.Hash{})
	assert.False(t, acct.HasSelfDestructed())
	revert := acct.SelfDestruct()
	assert.True(t, acct.HasSelfDestructed())
	revert()
	assert.False(t, acct.HasSelfDestructed())

	acct.SelfDestruct()
	assert.True(t, acct.HasSelfDestructed())
	assert.Equal(t, uint64(2), acct.Version())
	revert = acct.SelfDestruct()
	assert.True(t, acct.HasSelfDestructed())
	assert.Equal(t, uint64(2), acct.Version())
	revert()
	assert.True(t, acct.HasSelfDestructed())
	assert.Equal(t, uint64(2), acct.Version())
}

func TestGetFinalized(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: true,
			Version:      1,
		},
		nil,
		map[common.Hash]common.Hash{
			common.HexToHash("0x1"): common.HexToHash("0x11"),
		})
	acct.SetBalance(uint256.NewInt(10))
	acct.SetCode([]byte{1, 2, 3})
	acct.SetNonce(1)
	acct.SetState(common.HexToHash("0x1"), common.HexToHash("0x12"))
	acct.SetState(common.HexToHash("0x2"), common.HexToHash("0x22"))

	finalState, knownCode, knownCodeDiff, knownStorage, overriden := acct.GetFinalized()
	assert.Equal(t, uint64(1), finalState.Nonce)
	assert.Equal(t, uint256.NewInt(10), finalState.Balance)
	assert.Equal(t, crypto.Keccak256Hash([]byte{1, 2, 3}), finalState.CodeHash)
	assert.Equal(t, true, finalState.DirtyStorage)
	assert.Equal(t, uint64(1), finalState.Version)
	assert.Equal(t, []byte{1, 2, 3}, knownCode)
	assert.Equal(t, 1, len(knownCodeDiff))
	assert.Equal(t, int64(1), knownCodeDiff[crypto.Keccak256Hash([]byte{1, 2, 3})])
	assert.Equal(t, 2, len(knownStorage))
	assert.Equal(t, common.HexToHash("0x12"), knownStorage[common.HexToHash("0x1")])
	assert.Equal(t, common.HexToHash("0x22"), knownStorage[common.HexToHash("0x2")])
	assert.Equal(t, false, overriden)

	acct.SelfDestruct()
	finalState, knownCode, knownCodeDiff, knownStorage, overriden = acct.GetFinalized()
	assert.Equal(t, uint64(0), finalState.Nonce)
	assert.Equal(t, uint256.NewInt(0), finalState.Balance)
	assert.Equal(t, types.EmptyCodeHash, finalState.CodeHash)
	assert.Equal(t, false, finalState.DirtyStorage)
	assert.Equal(t, uint64(3), finalState.Version)
	assert.Nil(t, knownCode)
	assert.Equal(t, 1, len(knownCodeDiff))
	assert.Equal(t, int64(0), knownCodeDiff[crypto.Keccak256Hash([]byte{1, 2, 3})])
	assert.Equal(t, 0, len(knownStorage))
	assert.Equal(t, false, overriden)
}

func TestCopy(t *testing.T) {
	acct := newMutableAccountImpl(
		nil,
		common.HexToAddress(testAcctStr),
		itypes.AccountValue{
			Nonce:        0,
			Balance:      uint256.NewInt(0),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: true,
			Version:      1,
		},
		nil,
		map[common.Hash]common.Hash{
			common.HexToHash("0x1"): common.HexToHash("0x11"),
		})
	acct.SetBalance(uint256.NewInt(10))
	acct.SetCode([]byte{1, 2, 3})
	acct.SetNonce(1)
	acct.SetState(common.HexToHash("0x1"), common.HexToHash("0x12"))
	acct.SetState(common.HexToHash("0x2"), common.HexToHash("0x22"))

	acct2 := acct.Copy()
	finalState, knownCode, knownCodeDiff, knownStorage, overriden := acct.GetFinalized()
	finalState2, knownCode2, knownCodeDiff2, knownStorage2, overriden2 := acct2.GetFinalized()
	assert.Equal(t, finalState.Nonce, finalState2.Nonce)
	assert.Equal(t, finalState.Balance, finalState2.Balance)
	assert.Equal(t, finalState.CodeHash, finalState2.CodeHash)
	assert.Equal(t, finalState.DirtyStorage, finalState2.DirtyStorage)
	assert.Equal(t, finalState.Version, finalState2.Version)
	assert.Equal(t, knownCode, knownCode2)
	assert.Equal(t, len(knownCodeDiff), len(knownCodeDiff2))
	assert.Equal(t, int64(1), knownCodeDiff[crypto.Keccak256Hash([]byte{1, 2, 3})])
	assert.Equal(t, int64(1), knownCodeDiff2[crypto.Keccak256Hash([]byte{1, 2, 3})])
	assert.Equal(t, len(knownStorage), len(knownStorage2))
	assert.Equal(t, common.HexToHash("0x12"), knownStorage[common.HexToHash("0x1")])
	assert.Equal(t, common.HexToHash("0x22"), knownStorage[common.HexToHash("0x2")])
	assert.Equal(t, common.HexToHash("0x12"), knownStorage2[common.HexToHash("0x1")])
	assert.Equal(t, common.HexToHash("0x22"), knownStorage2[common.HexToHash("0x2")])
	assert.Equal(t, overriden, overriden2)
}
