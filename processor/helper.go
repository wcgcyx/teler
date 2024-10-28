package processor

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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/wcgcyx/teler/blockchain"
	"github.com/wcgcyx/teler/worldstate"
)

var (
	beaconDifficulty = common.Big0 // The default block difficulty in the beacon consensus
	// Some weird constants to avoid constant memory allocs for them.
	u256_8  = uint256.NewInt(8)
	u256_32 = uint256.NewInt(32)
)

type chainContext struct {
	ctx    context.Context
	chain  blockchain.Blockchain
	engine consensus.Engine
}

// NewChainContext creates a new chain context
func NewChainContext(ctx context.Context, chain blockchain.Blockchain, engine consensus.Engine) core.ChainContext {
	return &chainContext{
		ctx:    ctx,
		chain:  chain,
		engine: engine,
	}
}

// Engine retrieves the chain's consensus engine.
func (c *chainContext) Engine() consensus.Engine {
	return c.engine
}

// GetHeader returns the header corresponding to the hash/number argument pair.
func (c *chainContext) GetHeader(hash common.Hash, height uint64) *types.Header {
	header, err := c.chain.GetHeaderByHash(c.ctx, hash)
	if err != nil {
		log.Errorf("Fail to get header by hash for %v-%v: %v", height, hash, err.Error())
		header, err = c.chain.GetHeaderByNumber(c.ctx, height)
		if err != nil {
			log.Errorf("Fail to get header by number for %v-%v: %v", height, hash, err.Error())
		}
	}
	return header
}

// applyDAOHardFork modifies the state database according to the DAO hard-fork
// rules, transferring all balances of a set of DAO accounts to a single refund
// contract.
func applyDAOHardFork(worldState worldstate.WorldState) {
	// Retrieve the contract to refund balances into
	if !worldState.Exist(params.DAORefundContract) {
		worldState.CreateAccount(params.DAORefundContract)
	}

	// Move every DAO account and extra-balance account funds into the refund contract
	for _, addr := range params.DAODrainList() {
		worldState.AddBalance(params.DAORefundContract, worldState.GetBalance(addr), tracing.BalanceIncreaseDaoContract)
		worldState.SetBalance(addr, new(uint256.Int), tracing.BalanceDecreaseDaoAccount)
	}
}

// processBeaconBlockRoot applies the EIP-4788 system call to the beacon block root
// contract. This method is exported to be used in tests.
func processBeaconBlockRoot(beaconRoot common.Hash, vmenv *vm.EVM, worldState worldstate.WorldState) {
	// If EIP-4788 is enabled, we need to invoke the beaconroot storage contract with
	// the new root
	msg := &core.Message{
		From:      params.SystemAddress,
		GasLimit:  30_000_000,
		GasPrice:  common.Big0,
		GasFeeCap: common.Big0,
		GasTipCap: common.Big0,
		To:        &params.BeaconRootsAddress,
		Data:      beaconRoot[:],
	}
	vmenv.Reset(core.NewEVMTxContext(msg), worldState)
	worldState.AddAddressToAccessList(params.BeaconRootsAddress)
	_, _, _ = vmenv.Call(vm.AccountRef(msg.From), *msg.To, msg.Data, 30_000_000, common.U2560)
	worldState.Finalise(true)
}

// applyTransaction applies a transaction to state.
func applyTransaction(msg *core.Message, config *params.ChainConfig, gp *core.GasPool, worldState worldstate.WorldState, blockNumber *big.Int, blockHash common.Hash, tx *types.Transaction, usedGas *uint64, evm *vm.EVM) (*types.Receipt, error) {
	// Create a new context to be used in the EVM environment.
	txContext := core.NewEVMTxContext(msg)
	evm.Reset(txContext, worldState)

	// Apply the transaction to the current state (included in the env).
	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, err
	}

	// Update the state with pending changes.
	var root []byte
	if config.IsByzantium(blockNumber) {
		worldState.Finalise(true)
	} else {
		root = worldState.IntermediateRoot(config.IsEIP158(blockNumber)).Bytes()
	}
	*usedGas += result.UsedGas

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: *usedGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas

	if tx.Type() == types.BlobTxType {
		receipt.BlobGasUsed = uint64(len(tx.BlobHashes()) * params.BlobTxBlobGasPerBlob)
		receipt.BlobGasPrice = evm.Context.BlobBaseFee
	}

	// If the transaction created a contract, store the creation address in the receipt.
	if msg.To == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
	}

	// Set the receipt logs and create the bloom filter.
	receipt.Logs = worldState.GetLogs(tx.Hash(), blockNumber.Uint64(), blockHash)
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(worldState.TxIndex())
	return receipt, err
}

func consensusEngineFinalize(config *params.ChainConfig, header *types.Header, worldState worldstate.WorldState, uncles []*types.Header, withdrawals []*types.Withdrawal) {
	if !isPoSHeader(header) {
		// Accumulate any block and uncle rewards
		accumulateRewards(config, worldState, header, uncles)
	} else {
		// Withdrawals processing.
		for _, w := range withdrawals {
			// Convert amount from gwei to wei.
			amount := new(uint256.Int).SetUint64(w.Amount)
			amount = amount.Mul(amount, uint256.NewInt(params.GWei))
			worldState.AddBalance(w.Address, amount, tracing.BalanceIncreaseWithdrawal)
		}
		// No block reward which is issued by consensus layer instead.
	}
}

// isPoSHeader reports the header belongs to the PoS-stage with some special fields.
// This function is not suitable for a part of APIs like Prepare or CalcDifficulty
// because the header difficulty is not set yet.
func isPoSHeader(header *types.Header) bool {
	if header.Difficulty == nil {
		panic("IsPoSHeader called with invalid difficulty")
	}
	return header.Difficulty.Cmp(beaconDifficulty) == 0
}

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
func accumulateRewards(config *params.ChainConfig, state worldstate.WorldState, header *types.Header, uncles []*types.Header) {
	// Select the correct block reward based on chain progression
	blockReward := ethash.FrontierBlockReward
	if config.IsByzantium(header.Number) {
		blockReward = ethash.ByzantiumBlockReward
	}
	if config.IsConstantinople(header.Number) {
		blockReward = ethash.ConstantinopleBlockReward
	}
	// Accumulate the rewards for the miner and any included uncles
	reward := new(uint256.Int).Set(blockReward)
	r := new(uint256.Int)
	hNum, _ := uint256.FromBig(header.Number)
	for _, uncle := range uncles {
		uNum, _ := uint256.FromBig(uncle.Number)
		r.AddUint64(uNum, 8)
		r.Sub(r, hNum)
		r.Mul(r, blockReward)
		r.Div(r, u256_8)
		state.AddBalance(uncle.Coinbase, r, tracing.BalanceIncreaseRewardMineUncle)

		r.Div(blockReward, u256_32)
		reward.Add(reward, r)
	}
	state.AddBalance(header.Coinbase, reward, tracing.BalanceIncreaseRewardMineBlock)
}
