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
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	itypes "github.com/wcgcyx/teler/types"
)

const (
	testDS       = "./test-ds"
	testAcct1Str = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	testAcct2Str = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
)

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	m.Run()
}

func TestNewStateStore(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	// Empty path should fail
	_, err := NewStateStoreImpl(ctx, Opts{
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, nil, common.Hash{})
	assert.NotNil(t, err)

	// Should fail if starting from genesis with empty genesis
	_, err = NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, nil, common.Hash{})
	assert.NotNil(t, err)

	// Start from genesis should work
	genesis := core.DefaultGenesisBlock()
	sstore, err := NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, genesis, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	sstore.Shutdown()

	// Open existing should work
	sstore, err = NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, nil, common.Hash{})
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	defer sstore.Shutdown()

	height, hash, err := sstore.GetPersistedHeight()
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), height)
	assert.Equal(t, genesis.ToBlock().Root(), hash)
}

func TestPutAndDeleteLayerLog(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGenesisBlock()
	sstore, err := NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, genesis, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	defer sstore.Shutdown()

	expected := itypes.LayerLogFromGenesis(genesis, genesis.ToBlock().Root())

	_, err = sstore.GetLayerLog(0, genesis.ToBlock().Root())
	assert.NotNil(t, err)

	txn, err := sstore.NewTransaction()
	assert.Nil(t, err)
	err = txn.PutLayerLog(*expected)
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)

	stored, err := sstore.GetLayerLog(0, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.True(t, reflect.DeepEqual(stored, *expected))

	txn, err = sstore.NewTransaction()
	assert.Nil(t, err)
	err = txn.DeleteLayerLog(0, genesis.ToBlock().Root())
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)

	_, err = sstore.GetLayerLog(0, genesis.ToBlock().Root())
	assert.NotNil(t, err)
}

func TestPutAndDeleteChildren(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGenesisBlock()
	sstore, err := NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, genesis, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	defer sstore.Shutdown()

	children, err := sstore.GetChildren(0, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.Equal(t, 0, len(children))

	txn, err := sstore.NewTransaction()
	assert.Nil(t, err)
	err = txn.PutChildren(0, genesis.ToBlock().Root(), []common.Hash{common.HexToHash("0x1"), common.HexToHash("0x2")})
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)

	children, err = sstore.GetChildren(0, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.Equal(t, []common.Hash{common.HexToHash("0x1"), common.HexToHash("0x2")}, children)

	txn, err = sstore.NewTransaction()
	assert.Nil(t, err)
	err = txn.DeleteChildren(0, genesis.ToBlock().Root())
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)

	_, err = sstore.GetChildren(0, genesis.ToBlock().Root())
	assert.NotNil(t, err)
}

func TestPersistLayerWithUpdatedAccounts(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGenesisBlock()
	sstore, err := NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, genesis, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	defer sstore.Shutdown()

	acct1, err := sstore.GetAccountValue(common.HexToAddress(testAcct1Str))
	assert.Nil(t, err)
	assert.Equal(t, itypes.AccountValue{
		Nonce:        0,
		Balance:      uint256.NewInt(0),
		CodeHash:     types.EmptyCodeHash,
		DirtyStorage: false,
		Version:      0,
	}, acct1)
	acct2, err := sstore.GetAccountValue(common.HexToAddress(testAcct2Str))
	assert.Nil(t, err)
	assert.Equal(t, itypes.AccountValue{
		Nonce:        0,
		Balance:      uint256.NewInt(0),
		CodeHash:     types.EmptyCodeHash,
		DirtyStorage: false,
		Version:      0,
	}, acct2)

	// Persist first layer
	txn, err := sstore.NewTransaction()
	assert.Nil(t, err)

	acctp1 := itypes.AccountValue{
		Nonce:        1,
		Balance:      uint256.NewInt(2),
		CodeHash:     common.HexToHash("0x3"),
		DirtyStorage: true,
		Version:      1,
	}

	acctp2 := itypes.AccountValue{
		Nonce:        2,
		Balance:      uint256.NewInt(4),
		CodeHash:     common.HexToHash("0x6"),
		DirtyStorage: false,
		Version:      1,
	}

	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber: 1,
		RootHash:    common.HexToHash("0x1"),
		UpdatedAccounts: map[common.Address]itypes.AccountValue{
			common.HexToAddress(testAcct1Str): acctp1,
			common.HexToAddress(testAcct2Str): acctp2,
		},
		UpdatedCodeCount: make(map[common.Hash]int64),
		CodePreimage:     make(map[common.Hash][]byte),
		UpdatedStorage:   make(map[string]map[common.Hash]common.Hash),
	})
	assert.Nil(t, err)

	err = txn.Commit()
	assert.Nil(t, err)

	acct1, err = sstore.GetAccountValue(common.HexToAddress(testAcct1Str))
	assert.Nil(t, err)
	assert.Equal(t, acctp1, acct1)
	acct2, err = sstore.GetAccountValue(common.HexToAddress(testAcct2Str))
	assert.Nil(t, err)
	assert.Equal(t, acctp2, acct2)
}

func TestPersistLayerWithDeletedAccounts(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGenesisBlock()
	sstore, err := NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, genesis, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	defer sstore.Shutdown()

	// Persist first layer with new account
	txn, err := sstore.NewTransaction()
	assert.Nil(t, err)

	acctp1 := itypes.AccountValue{
		Nonce:        1,
		Balance:      uint256.NewInt(2),
		CodeHash:     common.HexToHash("0x3"),
		DirtyStorage: true,
		Version:      1,
	}
	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber: 1,
		RootHash:    common.HexToHash("0x1"),
		UpdatedAccounts: map[common.Address]itypes.AccountValue{
			common.HexToAddress(testAcct1Str): acctp1,
		},
		UpdatedCodeCount: make(map[common.Hash]int64),
		CodePreimage:     make(map[common.Hash][]byte),
		UpdatedStorage:   make(map[string]map[common.Hash]common.Hash),
	})
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)

	acct1, err := sstore.GetAccountValue(common.HexToAddress(testAcct1Str))
	assert.Nil(t, err)
	assert.Equal(t, acctp1, acct1)

	// Persist second layer with deleted account
	txn, err = sstore.NewTransaction()
	assert.Nil(t, err)

	acctp1 = itypes.AccountValue{
		Nonce:        1,
		Balance:      uint256.NewInt(2),
		CodeHash:     common.HexToHash("0x3"),
		DirtyStorage: true,
		Version:      3,
	}
	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber: 2,
		RootHash:    common.HexToHash("0x2"),
		UpdatedAccounts: map[common.Address]itypes.AccountValue{
			common.HexToAddress(testAcct1Str): acctp1,
		},
		UpdatedCodeCount: make(map[common.Hash]int64),
		CodePreimage:     make(map[common.Hash][]byte),
		UpdatedStorage:   make(map[string]map[common.Hash]common.Hash),
	})
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)

	acct1, err = sstore.GetAccountValue(common.HexToAddress(testAcct1Str))
	assert.Nil(t, err)
	assert.Equal(t, itypes.AccountValue{
		Nonce:        0,
		Balance:      uint256.NewInt(0),
		CodeHash:     types.EmptyCodeHash,
		DirtyStorage: false,
		Version:      3,
	}, acct1)
}

func TestPersistLayerWithSkippableAccount(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGenesisBlock()
	sstore, err := NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, genesis, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	defer sstore.Shutdown()

	acct1, err := sstore.GetAccountValue(common.HexToAddress(testAcct1Str))
	assert.Nil(t, err)
	assert.Equal(t, itypes.AccountValue{
		Nonce:        0,
		Balance:      uint256.NewInt(0),
		CodeHash:     types.EmptyCodeHash,
		DirtyStorage: false,
		Version:      0,
	}, acct1)

	txn, err := sstore.NewTransaction()
	assert.Nil(t, err)

	acctp1 := itypes.AccountValue{
		Nonce:        1,
		Balance:      uint256.NewInt(2),
		CodeHash:     common.HexToHash("0x3"),
		DirtyStorage: true,
		Version:      0,
	}

	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber: 1,
		RootHash:    common.HexToHash("0x1"),
		UpdatedAccounts: map[common.Address]itypes.AccountValue{
			common.HexToAddress(testAcct1Str): acctp1,
		},
		UpdatedCodeCount: make(map[common.Hash]int64),
		CodePreimage:     make(map[common.Hash][]byte),
		UpdatedStorage:   make(map[string]map[common.Hash]common.Hash),
	})
	assert.Nil(t, err)

	err = txn.Commit()
	assert.Nil(t, err)

	acct1, err = sstore.GetAccountValue(common.HexToAddress(testAcct1Str))
	assert.Nil(t, err)
	assert.Equal(t, itypes.AccountValue{
		Nonce:        0,
		Balance:      uint256.NewInt(0),
		CodeHash:     types.EmptyCodeHash,
		DirtyStorage: false,
		Version:      0,
	}, acct1)
}

func TestPerestLayerWithNewCode(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGenesisBlock()
	sstore, err := NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, genesis, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	defer sstore.Shutdown()

	_, err = sstore.GetCodeByHash(common.HexToHash("0x3"))
	assert.NotNil(t, err)

	// Persist first layer
	txn, err := sstore.NewTransaction()
	assert.Nil(t, err)

	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber:     1,
		RootHash:        common.HexToHash("0x1"),
		UpdatedAccounts: make(map[common.Address]itypes.AccountValue),
		UpdatedCodeCount: map[common.Hash]int64{
			common.HexToHash("0x3"): 1,
		},
		CodePreimage: map[common.Hash][]byte{
			common.HexToHash("0x3"): {1, 2, 3},
		},
		UpdatedStorage: make(map[string]map[common.Hash]common.Hash),
	})
	assert.Nil(t, err)

	err = txn.Commit()
	assert.Nil(t, err)

	code, err := sstore.GetCodeByHash(common.HexToHash("0x3"))
	assert.Nil(t, err)
	assert.Equal(t, []byte{1, 2, 3}, code)
}

func TestPersistLayerWithDiffToCode(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGenesisBlock()
	sstore, err := NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, genesis, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	defer sstore.Shutdown()

	// Persist first layer
	txn, err := sstore.NewTransaction()
	assert.Nil(t, err)
	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber:     1,
		RootHash:        common.HexToHash("0x1"),
		UpdatedAccounts: make(map[common.Address]itypes.AccountValue),
		UpdatedCodeCount: map[common.Hash]int64{
			common.HexToHash("0x3"): 1,
		},
		CodePreimage: map[common.Hash][]byte{
			common.HexToHash("0x3"): {1, 2, 3},
		},
		UpdatedStorage: make(map[string]map[common.Hash]common.Hash),
	})
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)
	code, err := sstore.GetCodeByHash(common.HexToHash("0x3"))
	assert.Nil(t, err)
	assert.Equal(t, []byte{1, 2, 3}, code)

	// Persist second layer
	txn, err = sstore.NewTransaction()
	assert.Nil(t, err)
	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber:     2,
		RootHash:        common.HexToHash("0x2"),
		UpdatedAccounts: make(map[common.Address]itypes.AccountValue),
		UpdatedCodeCount: map[common.Hash]int64{
			common.HexToHash("0x3"): 1,
		},
		UpdatedStorage: make(map[string]map[common.Hash]common.Hash),
	})
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)
	code, err = sstore.GetCodeByHash(common.HexToHash("0x3"))
	assert.Nil(t, err)
	assert.Equal(t, []byte{1, 2, 3}, code)

	// Persist third layer
	txn, err = sstore.NewTransaction()
	assert.Nil(t, err)
	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber:     2,
		RootHash:        common.HexToHash("0x2"),
		UpdatedAccounts: make(map[common.Address]itypes.AccountValue),
		UpdatedCodeCount: map[common.Hash]int64{
			common.HexToHash("0x3"): -1,
		},
		UpdatedStorage: make(map[string]map[common.Hash]common.Hash),
	})
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)
	code, err = sstore.GetCodeByHash(common.HexToHash("0x3"))
	assert.Nil(t, err)
	assert.Equal(t, []byte{1, 2, 3}, code)

	// Persist fourth layer
	txn, err = sstore.NewTransaction()
	assert.Nil(t, err)
	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber:     2,
		RootHash:        common.HexToHash("0x2"),
		UpdatedAccounts: make(map[common.Address]itypes.AccountValue),
		UpdatedCodeCount: map[common.Hash]int64{
			common.HexToHash("0x3"): -1,
		},
		UpdatedStorage: make(map[string]map[common.Hash]common.Hash),
	})
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)
	_, err = sstore.GetCodeByHash(common.HexToHash("0x3"))
	assert.NotNil(t, err)
}

func TestPersistLayerWithZeroDiff(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGenesisBlock()
	sstore, err := NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, genesis, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	defer sstore.Shutdown()

	txn, err := sstore.NewTransaction()
	assert.Nil(t, err)
	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber:     1,
		RootHash:        common.HexToHash("0x1"),
		UpdatedAccounts: make(map[common.Address]itypes.AccountValue),
		UpdatedCodeCount: map[common.Hash]int64{
			common.HexToHash("0x3"): 0,
		},
		UpdatedStorage: make(map[string]map[common.Hash]common.Hash),
	})
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)
	_, err = sstore.GetCodeByHash(common.HexToHash("0x3"))
	assert.NotNil(t, err)
}

func TestPersistLayerWithNonExistingCodeAndNegativeDiff(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGenesisBlock()
	sstore, err := NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, genesis, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	defer sstore.Shutdown()

	txn, err := sstore.NewTransaction()
	assert.Nil(t, err)
	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber:     1,
		RootHash:        common.HexToHash("0x1"),
		UpdatedAccounts: make(map[common.Address]itypes.AccountValue),
		UpdatedCodeCount: map[common.Hash]int64{
			common.HexToHash("0x3"): -1,
		},
		UpdatedStorage: make(map[string]map[common.Hash]common.Hash),
	})
	assert.NotNil(t, err)
}

func TestPersistLayerWithNonExistingCodeAndPositiveDiff(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGenesisBlock()
	sstore, err := NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, genesis, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	defer sstore.Shutdown()

	txn, err := sstore.NewTransaction()
	assert.Nil(t, err)
	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber:     1,
		RootHash:        common.HexToHash("0x1"),
		UpdatedAccounts: make(map[common.Address]itypes.AccountValue),
		UpdatedCodeCount: map[common.Hash]int64{
			common.HexToHash("0x3"): 1,
		},
		UpdatedStorage: make(map[string]map[common.Hash]common.Hash),
	})
	assert.NotNil(t, err)
}

func TestPersistLayerWithExistingCodeAndTooSmallDiff(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGenesisBlock()
	sstore, err := NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, genesis, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	defer sstore.Shutdown()

	// Persist first layer
	txn, err := sstore.NewTransaction()
	assert.Nil(t, err)
	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber:     1,
		RootHash:        common.HexToHash("0x1"),
		UpdatedAccounts: make(map[common.Address]itypes.AccountValue),
		UpdatedCodeCount: map[common.Hash]int64{
			common.HexToHash("0x3"): 1,
		},
		CodePreimage: map[common.Hash][]byte{
			common.HexToHash("0x3"): {1, 2, 3},
		},
		UpdatedStorage: make(map[string]map[common.Hash]common.Hash),
	})
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)
	code, err := sstore.GetCodeByHash(common.HexToHash("0x3"))
	assert.Nil(t, err)
	assert.Equal(t, []byte{1, 2, 3}, code)

	// Persist second layer
	txn, err = sstore.NewTransaction()
	assert.Nil(t, err)
	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber:     2,
		RootHash:        common.HexToHash("0x2"),
		UpdatedAccounts: make(map[common.Address]itypes.AccountValue),
		UpdatedCodeCount: map[common.Hash]int64{
			common.HexToHash("0x3"): -2,
		},
		UpdatedStorage: make(map[string]map[common.Hash]common.Hash),
	})
	assert.NotNil(t, err)
}

func TestPersistLayerWithUpdatedStorage(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGenesisBlock()
	sstore, err := NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Minute,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, genesis, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	defer sstore.Shutdown()

	addr1 := common.HexToAddress(testAcct1Str)
	addr2 := common.HexToAddress(testAcct2Str)

	val, err := sstore.GetStorageByVersion(addr1, 1, common.HexToHash("0x1"))
	assert.Nil(t, err)
	assert.Equal(t, common.Hash{}, val)
	val, err = sstore.GetStorageByVersion(addr2, 3, common.HexToHash("0x2"))
	assert.Nil(t, err)
	assert.Equal(t, common.Hash{}, val)

	// Persist first layer
	txn, err := sstore.NewTransaction()
	assert.Nil(t, err)

	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber:      1,
		RootHash:         common.HexToHash("0x1"),
		UpdatedAccounts:  make(map[common.Address]itypes.AccountValue),
		UpdatedCodeCount: make(map[common.Hash]int64),
		CodePreimage:     make(map[common.Hash][]byte),
		UpdatedStorage: map[string]map[common.Hash]common.Hash{
			addr1.Hex() + "-1": {
				common.HexToHash("0x1"): common.HexToHash("0x11"),
			},
			addr2.Hex() + "-3": {
				common.HexToHash("0x2"): common.HexToHash("0x22"),
			},
		},
	})
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)

	val, err = sstore.GetStorageByVersion(addr1, 1, common.HexToHash("0x1"))
	assert.Nil(t, err)
	assert.Equal(t, common.HexToHash("0x11"), val)
	val, err = sstore.GetStorageByVersion(addr2, 3, common.HexToHash("0x2"))
	assert.Nil(t, err)
	assert.Equal(t, common.HexToHash("0x22"), val)
}

func TestAccountStoragePruning(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGenesisBlock()
	sstore, err := NewStateStoreImpl(ctx, Opts{
		Path:         testDS,
		GCPeriod:     time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}, genesis, genesis.ToBlock().Root())
	assert.Nil(t, err)
	assert.NotNil(t, sstore)
	defer sstore.Shutdown()

	addr1 := common.HexToAddress(testAcct1Str)
	addr2 := common.HexToAddress(testAcct2Str)

	// Persist first layer with new accounts, code and storage
	txn, err := sstore.NewTransaction()
	assert.Nil(t, err)

	acctp1 := itypes.AccountValue{
		Nonce:        1,
		Balance:      uint256.NewInt(2),
		CodeHash:     common.HexToHash("0x3"),
		DirtyStorage: true,
		Version:      1,
	}

	acctp2 := itypes.AccountValue{
		Nonce:        2,
		Balance:      uint256.NewInt(4),
		CodeHash:     common.HexToHash("0x4"),
		DirtyStorage: true,
		Version:      1,
	}

	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber: 1,
		RootHash:    common.HexToHash("0x1"),
		UpdatedAccounts: map[common.Address]itypes.AccountValue{
			addr1: acctp1,
			addr2: acctp2,
		},
		UpdatedCodeCount: map[common.Hash]int64{
			common.HexToHash("0x3"): 1,
			common.HexToHash("0x4"): 1,
		},
		CodePreimage: map[common.Hash][]byte{
			common.HexToHash("0x3"): {1, 2, 3},
			common.HexToHash("0x4"): {1, 2, 3, 4},
		},
		UpdatedStorage: map[string]map[common.Hash]common.Hash{
			addr1.Hex() + "-1": {
				common.HexToHash("0x1"): common.HexToHash("0x11"),
				common.HexToHash("0x2"): common.HexToHash("0x22"),
				common.HexToHash("0x3"): common.HexToHash("0x33"),
			},
			addr2.Hex() + "-1": {
				common.HexToHash("0x4"): common.HexToHash("0x44"),
				common.HexToHash("0x5"): common.HexToHash("0x55"),
			},
		},
	})
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)

	val, err := sstore.GetStorageByVersion(addr1, 1, common.HexToHash("0x1"))
	assert.Nil(t, err)
	assert.Equal(t, common.HexToHash("0x11"), val)
	val, err = sstore.GetStorageByVersion(addr1, 1, common.HexToHash("0x2"))
	assert.Nil(t, err)
	assert.Equal(t, common.HexToHash("0x22"), val)
	val, err = sstore.GetStorageByVersion(addr1, 1, common.HexToHash("0x3"))
	assert.Nil(t, err)
	assert.Equal(t, common.HexToHash("0x33"), val)
	val, err = sstore.GetStorageByVersion(addr2, 1, common.HexToHash("0x4"))
	assert.Nil(t, err)
	assert.Equal(t, common.HexToHash("0x44"), val)
	val, err = sstore.GetStorageByVersion(addr2, 1, common.HexToHash("0x5"))
	assert.Nil(t, err)
	assert.Equal(t, common.HexToHash("0x55"), val)

	// Persist second layer which deletes two accounts
	txn, err = sstore.NewTransaction()
	assert.Nil(t, err)

	acctp1 = itypes.AccountValue{
		Nonce:        1,
		Balance:      uint256.NewInt(2),
		CodeHash:     common.HexToHash("0x3"),
		DirtyStorage: true,
		Version:      3,
	}

	acctp2 = itypes.AccountValue{
		Nonce:        2,
		Balance:      uint256.NewInt(4),
		CodeHash:     common.HexToHash("0x4"),
		DirtyStorage: true,
		Version:      3,
	}

	err = txn.PersistLayerLog(itypes.LayerLog{
		BlockNumber: 2,
		RootHash:    common.HexToHash("0x2"),
		UpdatedAccounts: map[common.Address]itypes.AccountValue{
			addr1: acctp1,
			addr2: acctp2,
		},
		UpdatedCodeCount: map[common.Hash]int64{
			common.HexToHash("0x3"): -1,
			common.HexToHash("0x4"): -1,
		},
		CodePreimage:   map[common.Hash][]byte{},
		UpdatedStorage: map[string]map[common.Hash]common.Hash{},
	})
	assert.Nil(t, err)
	err = txn.Commit()
	assert.Nil(t, err)

	time.Sleep(3 * time.Second)

	val, err = sstore.GetStorageByVersion(addr1, 1, common.HexToHash("0x1"))
	assert.Nil(t, err)
	assert.Equal(t, common.Hash{}, val)
	val, err = sstore.GetStorageByVersion(addr1, 1, common.HexToHash("0x2"))
	assert.Nil(t, err)
	assert.Equal(t, common.Hash{}, val)
	val, err = sstore.GetStorageByVersion(addr1, 1, common.HexToHash("0x3"))
	assert.Nil(t, err)
	assert.Equal(t, common.Hash{}, val)
	val, err = sstore.GetStorageByVersion(addr2, 1, common.HexToHash("0x4"))
	assert.Nil(t, err)
	assert.Equal(t, common.Hash{}, val)
	val, err = sstore.GetStorageByVersion(addr2, 1, common.HexToHash("0x5"))
	assert.Nil(t, err)
	assert.Equal(t, common.Hash{}, val)
}
