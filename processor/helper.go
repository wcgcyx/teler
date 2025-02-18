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

type chainContext struct {
	ctx    context.Context
	chain  blockchain.Blockchain
	engine consensus.Engine
	config *params.ChainConfig
}

// NewChainContext creates a new chain context
func NewChainContext(ctx context.Context, chain blockchain.Blockchain, engine consensus.Engine, config *params.ChainConfig) core.ChainContext {
	return &chainContext{
		ctx:    ctx,
		chain:  chain,
		engine: engine,
		config: config,
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

// Config returns the chain's configuration.
func (c *chainContext) Config() *params.ChainConfig {
	return c.config
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

// ApplyTransaction applies a transaction to state.
func ApplyTransaction(msg *core.Message, gp *core.GasPool, worldState worldstate.WorldState, blockNumber *big.Int, blockHash common.Hash, tx *types.Transaction, usedGas *uint64, evm *vm.EVM) (receipt *types.Receipt, err error) {
	if hooks := evm.Config.Tracer; hooks != nil {
		if hooks.OnTxStart != nil {
			hooks.OnTxStart(evm.GetVMContext(), tx, msg.From)
		}
		if hooks.OnTxEnd != nil {
			defer func() { hooks.OnTxEnd(receipt, err) }()
		}
	}
	// Apply the transaction to the current state (included in the env).
	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, err
	}
	// Update the state with pending changes.
	var root []byte
	if evm.ChainConfig().IsByzantium(blockNumber) {
		evm.StateDB.Finalise(true)
	} else {
		root = worldState.IntermediateRoot(evm.ChainConfig().IsEIP158(blockNumber)).Bytes()
	}
	*usedGas += result.UsedGas

	// Merge the tx-local access event into the "block-local" one, in order to collect
	// all values, so that the witness can be built.
	// TODO: Handle verkle trie
	// if worldState.GetTrie().IsVerkle() {
	// 	statedb.AccessEvents().Merge(evm.AccessEvents)
	// }

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt = &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: *usedGas}
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
	receipt.Bloom = types.CreateBloom(receipt)
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(worldState.TxIndex())
	return
}
