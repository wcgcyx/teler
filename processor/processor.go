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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/params"
	logging "github.com/ipfs/go-log"
	"github.com/wcgcyx/teler/blockchain"
	"github.com/wcgcyx/teler/worldstate"
)

// Logger
var log = logging.Logger("processor")

// Note:
// This is adapted from:
// 		go-ethereum@v1.14.8/core/state_processor.go
// Adapted code have been commented below.
// When merge from upstream, perform check from Process() function below.

type BlockProcessor struct {
	// Chain configuration
	config *params.ChainConfig
	// Blockchain
	chain blockchain.Blockchain
	// Consensus engine for block rewards
	Engine consensus.Engine
}

// NewBlockProcessor creates a new block processor.
func NewBlockProcessor(config *params.ChainConfig, chain blockchain.Blockchain) (*BlockProcessor, error) {
	// Note: Consensus engine is only used to calculate the block reward, so pass nil for db is fine as it is not required.
	engine, err := ethconfig.CreateConsensusEngine(config, nil)
	if err != nil {
		return nil, err
	}
	return &BlockProcessor{
		config: config,
		chain:  chain,
		Engine: engine,
	}, nil
}

// NewBlockProcessorWithEngine creates a new block processor.
func NewBlockProcessorWithEngine(config *params.ChainConfig, chain blockchain.Blockchain, engine consensus.Engine) *BlockProcessor {
	return &BlockProcessor{
		config: config,
		chain:  chain,
		Engine: engine,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *BlockProcessor) Process(ctx context.Context, block *types.Block, worldState worldstate.WorldState, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
	var (
		receipts    types.Receipts
		usedGas     = new(uint64)
		header      = block.Header()
		blockHash   = block.Hash()
		blockNumber = block.Number()
		allLogs     []*types.Log
		gp          = new(core.GasPool).AddGas(block.GasLimit())
	)
	// Mutate the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		// START //
		applyDAOHardFork(worldState)
		// END //
	}
	var (
		context vm.BlockContext
		signer  = types.MakeSigner(p.config, header.Number, header.Time)
	)
	// START //
	context = core.NewEVMBlockContext(header, NewChainContext(ctx, p.chain, p.Engine), nil)
	vmenv := vm.NewEVM(context, vm.TxContext{}, worldState, p.config, cfg)
	// END //
	if beaconRoot := block.BeaconRoot(); beaconRoot != nil {
		// START //
		ProcessBeaconBlockRoot(*beaconRoot, vmenv, worldState)
		// END //
	}
	// Iterate over and process the individual transactions
	var lastGas uint64 = 0
	// cc, err := ethclient.Dial("https://eth-mainnet.g.alchemy.com/v2/6-OEoIaepTc4jNGfZ21VkHfO5cxXUi94")
	// if err != nil {
	// 	return nil, nil, 0, err
	// }
	for i, tx := range block.Transactions() {
		msg, err := core.TransactionToMessage(tx, signer, header.BaseFee)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		// START //
		worldState.SetTxContext(tx.Hash(), i)
		receipt, err := ApplyTransaction(msg, p.config, gp, worldState, blockNumber, blockHash, tx, usedGas, vmenv)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		log.Debugf("Txn %v, Used gas: %v, status %v", tx.Hash(), *usedGas-lastGas, receipt.Status)
		// rr, err := cc.TransactionReceipt(ctx, tx.Hash())
		// if err != nil {
		// 	return nil, nil, 0, err
		// }
		// if rr.GasUsed != *usedGas-lastGas {
		// 	log.Infof("Tx %v local %v remote %v", tx.Hash(), *usedGas-lastGas, rr.GasUsed)
		// 	return nil, nil, 0, fmt.Errorf("fail now")
		// }
		lastGas = *usedGas
		// END //
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	// Fail if Shanghai not enabled and len(withdrawals) is non-zero.
	withdrawals := block.Withdrawals()
	if len(withdrawals) > 0 && !p.config.IsShanghai(block.Number(), block.Time()) {
		return nil, nil, 0, errors.New("withdrawals before shanghai")
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	// START
	consensusEngineFinalize(p.config, header, worldState, block.Uncles(), withdrawals)
	// END

	return receipts, allLogs, *usedGas, nil
}
