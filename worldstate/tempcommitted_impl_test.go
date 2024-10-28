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
	"github.com/wcgcyx/teler/statestore"
	itypes "github.com/wcgcyx/teler/types"
	"go.uber.org/mock/gomock"
)

func TestTempCommittedStateGetMutable(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockLayeredWorldState(ctrl)
	m.
		EXPECT().
		GetAccountValue(common.HexToAddress(testAcctStr), true).
		Return(itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(2),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		})

	committed := newTempCommittedState(nil, nil, m, 1)
	acct := committed.GetMutableAccount(common.HexToAddress(testAcctStr))
	assert.Equal(t, uint64(1), acct.Nonce())
	assert.Equal(t, uint256.NewInt(2), acct.Balance())
	assert.Equal(t, types.EmptyCodeHash, acct.CodeHash())
	assert.Equal(t, false, acct.DirtyStorage())
	assert.Equal(t, uint64(1), acct.Version())
}

func TestApplyChangeAndLogs(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockLayeredWorldState(ctrl)
	m.
		EXPECT().
		GetAccountValue(common.HexToAddress(testAcctStr), true).
		Return(itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(2),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		})

	committed := newTempCommittedState(nil, nil, m, 1)
	acct := committed.GetMutableAccount(common.HexToAddress(testAcctStr))
	acct.SetCode([]byte{1, 2, 3})
	acct.SetState(common.HexToHash("0x11"), common.HexToHash("0x111"))
	committed.ApplyChanges(
		common.HexToHash("0x1"),
		1,
		[]*types.Log{
			{Address: common.HexToAddress(testAcctStr)},
		},
		map[common.Address]mutableAccount{
			common.HexToAddress(testAcctStr): acct,
		})
	assert.Equal(t, uint(1), committed.GetLogSize())

	logs := committed.GetLogs(common.HexToHash("0x2"), 1, common.HexToHash("0x123"))
	assert.Equal(t, 0, len(logs))

	logs = committed.GetLogs(common.HexToHash("0x1"), 1, common.HexToHash("0x123"))
	assert.Equal(t, 1, len(logs))
	assert.Equal(t, uint64(1), logs[0].BlockNumber)
	assert.Equal(t, common.HexToHash("0x123"), logs[0].BlockHash)
}

func TestCommitChanges(t *testing.T) {
	ctrl := gomock.NewController(t)
	layer := itypes.LayerLog{
		BlockNumber: uint64(1),
		RootHash:    common.HexToHash("0x123"),
		UpdatedAccounts: map[common.Address]itypes.AccountValue{
			common.HexToAddress(testAcctStr): {
				Nonce:        1,
				Balance:      uint256.NewInt(2),
				CodeHash:     crypto.Keccak256Hash([]byte{1, 2, 3}),
				DirtyStorage: true,
				Version:      1,
			},
		},
		UpdatedCodeCount: map[common.Hash]int64{
			crypto.Keccak256Hash([]byte{1, 2, 3}): 1,
		},
		CodePreimage: map[common.Hash][]byte{
			crypto.Keccak256Hash([]byte{1, 2, 3}): {1, 2, 3},
		},
		UpdatedStorage: map[string]map[common.Hash]common.Hash{
			testAcctStr + "-1": {
				common.HexToHash("0x11"): common.HexToHash("0x111"),
			},
		},
	}
	txn := statestore.NewMockTransaction(ctrl)
	txn.
		EXPECT().
		PutChildren(uint64(1), common.HexToHash("0x123"), []common.Hash{}).
		Return(nil)
	txn.
		EXPECT().
		PutLayerLog(layer).
		Return(nil)
	txn.
		EXPECT().
		Commit().
		Return(nil)
	txn.
		EXPECT().
		Discard()
	m1 := statestore.NewMockStateStore(ctrl)
	m1.
		EXPECT().
		NewTransaction().
		Return(txn, nil)
	m1.
		EXPECT().
		GetLayerLog(uint64(1), common.HexToHash("0x123")).
		Return(layer, nil)
	m1.
		EXPECT().
		GetChildren(uint64(1), common.HexToHash("0x123")).
		Return([]common.Hash{}, nil)
	m2 := NewMockLayeredWorldStateArchive(ctrl)
	m2.
		EXPECT().
		Has(uint64(1), common.HexToHash("0x123")).
		Return(false)
	m2.
		EXPECT().
		Register(uint64(1), common.HexToHash("0x123"), gomock.Any())
	m3 := NewMockLayeredWorldState(ctrl)
	m3.
		EXPECT().
		GetAccountValue(common.HexToAddress(testAcctStr), true).
		Return(itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(2),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		})
	m3.
		EXPECT().
		AddChild(gomock.Any())
	m3.
		EXPECT().
		HandlePrune([]common.Hash{common.HexToHash("0x123")})

	committed := newTempCommittedState(m1, m2, m3, 1)
	acct := committed.GetMutableAccount(common.HexToAddress(testAcctStr))
	acct.SetCode([]byte{1, 2, 3})
	acct.SetState(common.HexToHash("0x11"), common.HexToHash("0x111"))
	committed.ApplyChanges(
		common.HexToHash("0x1"),
		1,
		[]*types.Log{
			{Address: common.HexToAddress(testAcctStr)},
		},
		map[common.Address]mutableAccount{
			common.HexToAddress(testAcctStr): acct,
		})

	res, err := committed.Commit(1, common.HexToHash("0x123"))
	assert.Nil(t, err)
	assert.Equal(t, common.HexToHash("0x123"), res)
}

func TestCommittedStateCopy(t *testing.T) {
	ctrl := gomock.NewController(t)
	m1 := NewMockLayeredWorldState(ctrl)
	m1.
		EXPECT().
		GetAccountValue(common.HexToAddress(testAcctStr), true).
		Return(itypes.AccountValue{
			Nonce:        1,
			Balance:      uint256.NewInt(2),
			CodeHash:     types.EmptyCodeHash,
			DirtyStorage: false,
			Version:      1,
		})
	m2 := NewMockLayeredWorldStateArchive(ctrl)

	committed := newTempCommittedState(nil, m2, m1, 1)
	acct := committed.GetMutableAccount(common.HexToAddress(testAcctStr))
	acct.SetCode([]byte{1, 2, 3})
	acct.SetState(common.HexToHash("0x11"), common.HexToHash("0x111"))
	committed.ApplyChanges(
		common.HexToHash("0x1"),
		1,
		[]*types.Log{
			{Address: common.HexToAddress(testAcctStr)},
		},
		map[common.Address]mutableAccount{
			common.HexToAddress(testAcctStr): acct,
		})
	committed2 := committed.Copy()
	assert.Equal(t, committed, committed2)
}
