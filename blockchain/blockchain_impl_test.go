package blockchain

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
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/stretchr/testify/assert"
)

const (
	testDS       = "./test-ds"
	testKey1     = "1111111111111111111111111111111111111111111111111111111111111111"
	testAcct1Str = "0x19E7E376E7C213B7E7e7e46cc70A5dD086DAff2A"
	testKey2     = "2222222222222222222222222222222222222222222222222222222222222222"
	testAcct2Str = "0x1563915e194D8CfBA1943570603F7606A3115508"
)

var (
	testAcct1 = common.HexToAddress(testAcct1Str)
	testAcct2 = common.HexToAddress(testAcct2Str)
)

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	m.Run()
}

func TestNewBlockchain(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGenesisBlock()
	// Empty path for genesis should fail
	_, err := NewBlockchainImpl(ctx, Opts{}, genesis)
	assert.NotNil(t, err)

	// Non empty path for genesis should succeed
	blockchain, err := NewBlockchainImpl(ctx, Opts{
		Path:             testDS,
		MaxBlockToRetain: 256,
		PruningFrequency: 128,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     5 * time.Second,
	}, genesis)
	assert.Nil(t, err)
	assert.NotNil(t, blockchain)
	// Close blockchain
	blockchain.Shutdown()

	// Open existing db with genesis should succeed.
	blockchain, err = NewBlockchainImpl(ctx, Opts{
		Path:             testDS,
		MaxBlockToRetain: 256,
		PruningFrequency: 128,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     5 * time.Second,
	}, genesis)
	assert.Nil(t, err)
	assert.NotNil(t, blockchain)
	// Close blockchain
	blockchain.Shutdown()
}

func TestAddBlockOfSingleChain(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGoerliGenesisBlock()
	genesis.Alloc[testAcct1] = types.Account{
		Code:    []byte{},
		Storage: make(map[common.Hash]common.Hash),
		Balance: big.NewInt(1000000000000000000),
		Nonce:   0,
	}
	blockchain, _ := NewBlockchainImpl(ctx, Opts{
		Path:             testDS,
		MaxBlockToRetain: 256,
		PruningFrequency: 128,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     5 * time.Second,
	}, genesis)
	assert.NotNil(t, blockchain)

	// Generate blocks
	engine, _ := ethconfig.CreateConsensusEngine(genesis.Config, rawdb.NewMemoryDatabase())
	_, blocks, receipts := core.GenerateChainWithGenesis(genesis, engine, 64, func(i int, bg *core.BlockGen) {
		for j := 0; j < rand.Intn(5); j++ {
			// Simple transfer from test account 1 to test account 2
			tx := types.NewTransaction(bg.TxNonce(testAcct1), testAcct2, big.NewInt(1), 21000, big.NewInt(1), nil)
			key, _ := crypto.HexToECDSA(testKey1)
			tx, _ = types.SignTx(tx, types.MakeSigner(genesis.Config, big.NewInt(int64(i)), bg.Timestamp()), key)
			bg.AddTx(tx)
		}
	})

	for i, blk := range blocks {
		err := blockchain.AddBlock(ctx, blk, receipts[i])
		assert.Nil(t, err)
	}

	err := blockchain.SetHead(ctx, blocks[63].Hash())
	assert.Nil(t, err)

	head, err := blockchain.GetHead(ctx)
	assert.Nil(t, err)
	assert.Equal(t, blocks[63].Hash(), head.Hash())

	blk, err := blockchain.GetBlockByNumber(ctx, 0)
	assert.Nil(t, err)
	assert.Equal(t, genesis.ToBlock().Hash(), blk.Hash())

	blk, err = blockchain.GetBlockByNumber(ctx, 32)
	assert.Nil(t, err)
	assert.Equal(t, blocks[31].Hash(), blk.Hash())

	blk, err = blockchain.GetBlockByNumber(ctx, 63)
	assert.Nil(t, err)
	assert.Equal(t, blocks[62].Hash(), blk.Hash())
}

func TestAddBlockOfForks(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGoerliGenesisBlock()
	genesis.Alloc[testAcct1] = types.Account{
		Code:    []byte{},
		Storage: make(map[common.Hash]common.Hash),
		Balance: big.NewInt(1000000000000000000),
		Nonce:   0,
	}
	blockchain, _ := NewBlockchainImpl(ctx, Opts{
		Path:             testDS,
		MaxBlockToRetain: 256,
		PruningFrequency: 128,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     5 * time.Second,
	}, genesis)
	assert.NotNil(t, blockchain)

	engine, _ := ethconfig.CreateConsensusEngine(genesis.Config, rawdb.NewMemoryDatabase())
	// Generate canonical chain
	db, blocks, receipts := core.GenerateChainWithGenesis(genesis, engine, 64, func(i int, bg *core.BlockGen) {
		for j := 0; j < rand.Intn(5); j++ {
			// Simple transfer from test account 1 to test account 2
			tx := types.NewTransaction(bg.TxNonce(testAcct1), testAcct2, big.NewInt(1), 21000, big.NewInt(1), nil)
			key, _ := crypto.HexToECDSA(testKey1)
			tx, _ = types.SignTx(tx, types.MakeSigner(genesis.Config, big.NewInt(int64(i)), bg.Timestamp()), key)
			bg.AddTx(tx)
		}
	})
	for i, blk := range blocks {
		err := blockchain.AddBlock(ctx, blk, receipts[i])
		assert.Nil(t, err)
	}

	// Generate fork
	blocks2, receipts2 := core.GenerateChain(genesis.Config, blocks[31], engine, db, 32, func(i int, bg *core.BlockGen) {
		for j := 0; j < rand.Intn(5); j++ {
			// Simple transfer from test account 1 to test account 2
			tx := types.NewTransaction(bg.TxNonce(testAcct1), testAcct2, big.NewInt(1), 21000, big.NewInt(1), nil)
			key, _ := crypto.HexToECDSA(testKey1)
			tx, _ = types.SignTx(tx, types.MakeSigner(genesis.Config, big.NewInt(int64(i)), bg.Timestamp()), key)
			bg.AddTx(tx)
		}
	})

	for i, blk := range blocks2 {
		err := blockchain.AddBlock(ctx, blk, receipts2[i])
		assert.Nil(t, err)
	}

	err := blockchain.SetHead(ctx, blocks[63].Hash())
	assert.Nil(t, err)

	head, err := blockchain.GetHead(ctx)
	assert.Nil(t, err)
	assert.Equal(t, blocks[63].Hash(), head.Hash())

	blk, err := blockchain.GetBlockByNumber(ctx, 0)
	assert.Nil(t, err)
	assert.Equal(t, genesis.ToBlock().Hash(), blk.Hash())

	blk, err = blockchain.GetBlockByNumber(ctx, 32)
	assert.Nil(t, err)
	assert.Equal(t, blocks[31].Hash(), blk.Hash())

	blk, err = blockchain.GetBlockByNumber(ctx, 63)
	assert.Nil(t, err)
	assert.Equal(t, blocks[62].Hash(), blk.Hash())

	expected := blocks2[31].Header().Hash()
	exists, err := blockchain.HasBlock(ctx, expected)
	assert.Nil(t, err)
	assert.True(t, exists)
	blk, err = blockchain.GetBlockByHash(ctx, expected)
	assert.Nil(t, err)
	assert.Equal(t, expected, blk.Hash())
}

func TestFinalizedAndSafe(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGoerliGenesisBlock()
	genesis.Alloc[testAcct1] = types.Account{
		Code:    []byte{},
		Storage: make(map[common.Hash]common.Hash),
		Balance: big.NewInt(1000000000000000000),
		Nonce:   0,
	}
	blockchain, _ := NewBlockchainImpl(ctx, Opts{
		Path:             testDS,
		MaxBlockToRetain: 256,
		PruningFrequency: 128,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     5 * time.Second,
	}, genesis)
	assert.NotNil(t, blockchain)

	// Generate blocks
	engine, _ := ethconfig.CreateConsensusEngine(genesis.Config, rawdb.NewMemoryDatabase())
	_, blocks, receipts := core.GenerateChainWithGenesis(genesis, engine, 64, func(i int, bg *core.BlockGen) {
		for j := 0; j < rand.Intn(5); j++ {
			// Simple transfer from test account 1 to test account 2
			tx := types.NewTransaction(bg.TxNonce(testAcct1), testAcct2, big.NewInt(1), 21000, big.NewInt(1), nil)
			key, _ := crypto.HexToECDSA(testKey1)
			tx, _ = types.SignTx(tx, types.MakeSigner(genesis.Config, big.NewInt(int64(i)), bg.Timestamp()), key)
			bg.AddTx(tx)
		}
	})

	for i, blk := range blocks {
		err := blockchain.AddBlock(ctx, blk, receipts[i])
		assert.Nil(t, err)
	}

	err := blockchain.SetHead(ctx, blocks[63].Hash())
	assert.Nil(t, err)

	head, err := blockchain.GetHead(ctx)
	assert.Nil(t, err)
	assert.Equal(t, blocks[63].Hash(), head.Hash())

	_, exists, err := blockchain.GetFinalized(ctx)
	assert.Nil(t, err)
	assert.False(t, exists)

	err = blockchain.SetFinalized(ctx, blocks[50].Hash())
	assert.Nil(t, err)

	finalized, exists, err := blockchain.GetFinalized(ctx)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, blocks[50].Hash(), finalized.Hash())

	err = blockchain.SetFinalized(ctx, blocks[60].Hash())
	assert.Nil(t, err)

	finalized, exists, err = blockchain.GetFinalized(ctx)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, blocks[60].Hash(), finalized.Hash())

	_, exists, err = blockchain.GetSafe(ctx)
	assert.Nil(t, err)
	assert.False(t, exists)

	err = blockchain.SetSafe(ctx, blocks[49].Hash())
	assert.Nil(t, err)

	safe, exists, err := blockchain.GetSafe(ctx)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, blocks[49].Hash(), safe.Hash())

	err = blockchain.SetSafe(ctx, blocks[61].Hash())
	assert.Nil(t, err)

	safe, exists, err = blockchain.GetSafe(ctx)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, blocks[61].Hash(), safe.Hash())
}

func TestGetBlockData(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGoerliGenesisBlock()
	genesis.Alloc[testAcct1] = types.Account{
		Code:    []byte{},
		Storage: make(map[common.Hash]common.Hash),
		Balance: big.NewInt(1000000000000000000),
		Nonce:   0,
	}
	blockchain, _ := NewBlockchainImpl(ctx, Opts{
		Path:             testDS,
		MaxBlockToRetain: 256,
		PruningFrequency: 128,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     5 * time.Second,
	}, genesis)
	assert.NotNil(t, blockchain)

	// Generate blocks
	engine, _ := ethconfig.CreateConsensusEngine(genesis.Config, rawdb.NewMemoryDatabase())
	_, blocks, receipts := core.GenerateChainWithGenesis(genesis, engine, 64, func(i int, bg *core.BlockGen) {
		for j := 0; j <= rand.Intn(5); j++ {
			// Simple transfer from test account 1 to test account 2
			tx := types.NewTransaction(bg.TxNonce(testAcct1), testAcct2, big.NewInt(1), 21000, big.NewInt(1), nil)
			key, _ := crypto.HexToECDSA(testKey1)
			tx, _ = types.SignTx(tx, types.MakeSigner(genesis.Config, big.NewInt(int64(i)), bg.Timestamp()), key)
			bg.AddTx(tx)
		}
	})
	// Fix, receipts have empty logs from chain generation, overwrite to empty list
	for _, rs := range receipts {
		for _, r := range rs {
			r.Logs = make([]*types.Log, 0)
		}
	}

	for i, blk := range blocks {
		err := blockchain.AddBlock(ctx, blk, receipts[i])
		assert.Nil(t, err)
	}

	err := blockchain.SetHead(ctx, blocks[63].Hash())
	assert.Nil(t, err)

	head, err := blockchain.GetHead(ctx)
	assert.Nil(t, err)
	assert.Equal(t, blocks[63].Hash(), head.Hash())

	blk, err := blockchain.GetHeaderByNumber(ctx, 0)
	assert.Nil(t, err)
	assert.Equal(t, genesis.ToBlock().Hash(), blk.Hash())

	for i, expected := range blocks[0].Transactions() {
		txn, blkHash, index, exists, err := blockchain.GetTransaction(ctx, expected.Hash())
		assert.Nil(t, err)
		assert.True(t, exists)
		assert.Equal(t, blocks[0].Hash(), blkHash)
		assert.Equal(t, uint64(i), index)
		assert.Equal(t, expected.Hash(), txn.Hash())

		receipt, blkHash, index, exists, err := blockchain.GetReceipt(ctx, expected.Hash())
		assert.Nil(t, err)
		assert.True(t, exists)
		assert.Equal(t, blocks[0].Hash(), blkHash)
		assert.Equal(t, uint64(i), index)
		expectedData, err := receipts[0][i].MarshalBinary()
		assert.Nil(t, err)
		actualData, err := receipt.MarshalBinary()
		assert.Nil(t, err)
		assert.Equal(t, expectedData, actualData)
	}

	blk, err = blockchain.GetHeaderByHash(ctx, blocks[31].Hash())
	assert.Nil(t, err)
	assert.Equal(t, blocks[31].Hash(), blk.Hash())

	_, err = blockchain.GetHeaderByNumber(ctx, 100)
	assert.NotNil(t, err)

	for i, expected := range blocks[32].Transactions() {
		txn, blkHash, index, exists, err := blockchain.GetTransaction(ctx, expected.Hash())
		assert.Nil(t, err)
		assert.True(t, exists)
		assert.Equal(t, blocks[32].Hash(), blkHash)
		assert.Equal(t, uint64(i), index)
		assert.Equal(t, expected.Hash(), txn.Hash())

		receipt, blkHash, index, exists, err := blockchain.GetReceipt(ctx, expected.Hash())
		assert.Nil(t, err)
		assert.True(t, exists)
		assert.Equal(t, blocks[32].Hash(), blkHash)
		assert.Equal(t, uint64(i), index)
		expectedData, err := receipts[32][i].MarshalBinary()
		assert.Nil(t, err)
		actualData, err := receipt.MarshalBinary()
		assert.Nil(t, err)
		assert.Equal(t, expectedData, actualData)
	}

	blk, err = blockchain.GetHeaderByNumber(ctx, 63)
	assert.Nil(t, err)
	assert.Equal(t, blocks[62].Hash(), blk.Hash())

	for i, expected := range blocks[63].Transactions() {
		txn, blkHash, index, exists, err := blockchain.GetTransaction(ctx, expected.Hash())
		assert.Nil(t, err)
		assert.True(t, exists)
		assert.Equal(t, blocks[63].Hash(), blkHash)
		assert.Equal(t, uint64(i), index)
		assert.Equal(t, expected.Hash(), txn.Hash())

		receipt, blkHash, index, exists, err := blockchain.GetReceipt(ctx, expected.Hash())
		assert.Nil(t, err)
		assert.True(t, exists)
		assert.Equal(t, blocks[63].Hash(), blkHash)
		assert.Equal(t, uint64(i), index)
		expectedData, err := receipts[63][i].MarshalBinary()
		assert.Nil(t, err)
		actualData, err := receipt.MarshalBinary()
		assert.Nil(t, err)
		assert.Equal(t, expectedData, actualData)
	}

	_, _, _, exists, err := blockchain.GetTransaction(ctx, common.Hash{})
	assert.Nil(t, err)
	assert.False(t, exists)

	_, _, _, exists, err = blockchain.GetReceipt(ctx, common.Hash{})
	assert.Nil(t, err)
	assert.False(t, exists)
}

func TestReorg(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGoerliGenesisBlock()
	genesis.Alloc[testAcct1] = types.Account{
		Code:    []byte{},
		Storage: make(map[common.Hash]common.Hash),
		Balance: big.NewInt(1000000000000000000),
		Nonce:   0,
	}
	blockchain, _ := NewBlockchainImpl(ctx, Opts{
		Path:             testDS,
		MaxBlockToRetain: 256,
		PruningFrequency: 128,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     5 * time.Second,
	}, genesis)
	assert.NotNil(t, blockchain)

	engine, _ := ethconfig.CreateConsensusEngine(genesis.Config, rawdb.NewMemoryDatabase())
	// Generate canonical chain
	db, blocks, receipts := core.GenerateChainWithGenesis(genesis, engine, 64, func(i int, bg *core.BlockGen) {
		for j := 0; j < rand.Intn(5); j++ {
			// Simple transfer from test account 1 to test account 2
			tx := types.NewTransaction(bg.TxNonce(testAcct1), testAcct2, big.NewInt(1), 21000, big.NewInt(1), nil)
			key, _ := crypto.HexToECDSA(testKey1)
			tx, _ = types.SignTx(tx, types.MakeSigner(genesis.Config, big.NewInt(int64(i)), bg.Timestamp()), key)
			bg.AddTx(tx)
		}
	})
	for i, blk := range blocks {
		err := blockchain.AddBlock(ctx, blk, receipts[i])
		assert.Nil(t, err)
	}

	// Generate fork
	blocks2, receipts2 := core.GenerateChain(genesis.Config, blocks[31], engine, db, 32, func(i int, bg *core.BlockGen) {
		for j := 0; j < rand.Intn(5); j++ {
			// Simple transfer from test account 1 to test account 2
			tx := types.NewTransaction(bg.TxNonce(testAcct1), testAcct2, big.NewInt(1), 21000, big.NewInt(1), nil)
			key, _ := crypto.HexToECDSA(testKey1)
			tx, _ = types.SignTx(tx, types.MakeSigner(genesis.Config, big.NewInt(int64(i)), bg.Timestamp()), key)
			bg.AddTx(tx)
		}
	})

	for i, blk := range blocks2 {
		err := blockchain.AddBlock(ctx, blk, receipts2[i])
		assert.Nil(t, err)
	}

	// First set head
	err := blockchain.SetHead(ctx, blocks[63].Hash())
	assert.Nil(t, err)

	head, err := blockchain.GetHead(ctx)
	assert.Nil(t, err)
	assert.Equal(t, blocks[63].Hash(), head.Hash())

	// Verify that we are on canonical chain.
	for i, expected := range blocks {
		tmp, err := blockchain.GetBlockByNumber(ctx, uint64(i+1))
		assert.Nil(t, err)
		assert.Equal(t, expected.Hash(), tmp.Hash())
	}

	// Reorg
	err = blockchain.SetHead(ctx, blocks2[31].Hash())
	assert.Nil(t, err)

	head, err = blockchain.GetHead(ctx)
	assert.Nil(t, err)
	assert.Equal(t, blocks2[31].Hash(), head.Hash())

	// Verify that we are on fork
	for i, expected := range blocks2 {
		tmp, err := blockchain.GetBlockByNumber(ctx, uint64(i+33))
		assert.Nil(t, err)
		assert.Equal(t, expected.Hash(), tmp.Hash())
	}

	// Reorg back
	err = blockchain.SetHead(ctx, blocks2[10].Hash())
	assert.Nil(t, err)

	head, err = blockchain.GetHead(ctx)
	assert.Nil(t, err)
	assert.Equal(t, blocks2[10].Hash(), head.Hash())

	// Verify that we are back
	for i, expected := range blocks2 {
		tmp, err := blockchain.GetBlockByNumber(ctx, uint64(i+33))
		if i <= 10 {
			assert.Nil(t, err)
			assert.Equal(t, expected.Hash(), tmp.Hash())
		} else {
			assert.NotNil(t, err)
		}
	}
}

func TestPruning(t *testing.T) {
	defer os.RemoveAll(testDS)

	ctx := context.Background()

	genesis := core.DefaultGoerliGenesisBlock()
	genesis.Alloc[testAcct1] = types.Account{
		Code:    []byte{},
		Storage: make(map[common.Hash]common.Hash),
		Balance: big.NewInt(1000000000000000000),
		Nonce:   0,
	}
	blockchain, _ := NewBlockchainImpl(ctx, Opts{
		Path:             testDS,
		MaxBlockToRetain: 256,
		PruningFrequency: 128,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     5 * time.Second,
	}, genesis)
	assert.NotNil(t, blockchain)

	engine, _ := ethconfig.CreateConsensusEngine(genesis.Config, rawdb.NewMemoryDatabase())
	// Generate canonical chain
	db, blocks, receipts := core.GenerateChainWithGenesis(genesis, engine, 500, func(i int, bg *core.BlockGen) {
		for j := 0; j < rand.Intn(5); j++ {
			// Simple transfer from test account 1 to test account 2
			tx := types.NewTransaction(bg.TxNonce(testAcct1), testAcct2, big.NewInt(1), 21000, big.NewInt(1), nil)
			key, _ := crypto.HexToECDSA(testKey1)
			tx, _ = types.SignTx(tx, types.MakeSigner(genesis.Config, big.NewInt(int64(i)), bg.Timestamp()), key)
			bg.AddTx(tx)
		}
	})
	// Generate fork
	blocks2, receipts2 := core.GenerateChain(genesis.Config, blocks[31], engine, db, 32, func(i int, bg *core.BlockGen) {
		for j := 0; j < rand.Intn(5); j++ {
			// Simple transfer from test account 1 to test account 2
			tx := types.NewTransaction(bg.TxNonce(testAcct1), testAcct2, big.NewInt(1), 21000, big.NewInt(1), nil)
			key, _ := crypto.HexToECDSA(testKey1)
			tx, _ = types.SignTx(tx, types.MakeSigner(genesis.Config, big.NewInt(int64(i)), bg.Timestamp()), key)
			bg.AddTx(tx)
		}
	})
	for i, blk := range blocks {
		err := blockchain.AddBlock(ctx, blk, receipts[i])
		assert.Nil(t, err)
	}
	for i, blk := range blocks2 {
		err := blockchain.AddBlock(ctx, blk, receipts2[i])
		assert.Nil(t, err)
	}

	// Without setting head, it won't prune
	for i := 0; i < 500; i++ {
		expected := blocks[i].Header().Hash()
		exists, err := blockchain.HasBlock(ctx, expected)
		assert.Nil(t, err)
		assert.True(t, exists)
	}
	for i := 0; i < 32; i++ {
		expected := blocks2[i].Header().Hash()
		exists, err := blockchain.HasBlock(ctx, expected)
		assert.Nil(t, err)
		assert.True(t, exists)
	}

	// Setting head to the newest block will prune old ones
	err := blockchain.SetHead(ctx, blocks[499].Hash())
	assert.Nil(t, err)

	for i := 0; i < 500; i++ {
		expected := blocks[i].Header().Hash()
		exists, err := blockchain.HasBlock(ctx, expected)
		assert.Nil(t, err)
		if i < 127 {
			assert.False(t, exists)
			for _, txn := range blocks[i].Transactions() {
				_, _, _, exists2, err2 := blockchain.GetTransaction(ctx, txn.Hash())
				assert.Nil(t, err2)
				assert.False(t, exists2)
			}
		} else {
			assert.True(t, exists)
		}
	}
	for i := 0; i < 32; i++ {
		expected := blocks2[i].Header().Hash()
		exists, err := blockchain.HasBlock(ctx, expected)
		assert.Nil(t, err)
		assert.False(t, exists)
	}
}
