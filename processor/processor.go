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

	"github.com/ethereum/go-ethereum/common"
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
// 		go-ethereum@v1.15.2/core/state_processor.go
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

// Config retrieves the blockchain's chain configuration.
func (p *BlockProcessor) Config() *params.ChainConfig {
	return p.config
}

// CurrentHeader retrieves the current header from the local chain.
func (p *BlockProcessor) CurrentHeader() *types.Header {
	blk, err := p.chain.GetHead(context.Background())
	if err != nil {
		log.Errorf("Fail to get current header in block processor: %v", err.Error())
		return &types.Header{}
	}
	return blk.Header()
}

// GetHeader retrieves a block header from the database by hash and number.
func (p *BlockProcessor) GetHeader(hash common.Hash, number uint64) *types.Header {
	res := p.GetHeaderByHash(hash)
	if res.Number.Uint64() != number {
		log.Errorf("Header number mismtach for %v, expect %v, got %v", hash, number, res.Number.Uint64())
		return &types.Header{}
	}
	return res
}

// GetHeaderByNumber retrieves a block header from the database by number.
func (p *BlockProcessor) GetHeaderByNumber(number uint64) *types.Header {
	res, err := p.chain.GetHeaderByNumber(context.Background(), number)
	if err != nil {
		log.Errorf("Fail to get current header by number in block processor: %v", err.Error())
		return &types.Header{}
	}
	return res
}

// GetHeaderByHash retrieves a block header from the database by its hash.
func (p *BlockProcessor) GetHeaderByHash(hash common.Hash) *types.Header {
	res, err := p.chain.GetHeaderByHash(context.Background(), hash)
	if err != nil {
		log.Errorf("Fail to get current header by hash in block processor: %v", err.Error())
		return &types.Header{}
	}
	return res
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
	context = core.NewEVMBlockContext(header, NewChainContext(ctx, p.chain, p.Engine, p.config), nil)
	evm := vm.NewEVM(context, worldState, p.config, cfg)
	// END //
	if beaconRoot := block.BeaconRoot(); beaconRoot != nil {
		// START //
		core.ProcessBeaconBlockRoot(*beaconRoot, evm)
		// END //
	}
	if p.config.IsPrague(block.Number(), block.Time()) || p.config.IsVerkle(block.Number(), block.Time()) {
		core.ProcessParentBlockHash(block.ParentHash(), evm)
	}

	// Iterate over and process the individual transactions
	var lastGas uint64 = 0
	for i, tx := range block.Transactions() {
		msg, err := core.TransactionToMessage(tx, signer, header.BaseFee)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		// START //
		worldState.SetTxContext(tx.Hash(), i)
		receipt, err := ApplyTransaction(msg, gp, worldState, blockNumber, blockHash, tx, usedGas, evm)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		log.Debugf("Txn %v, Used gas: %v, status %v", tx.Hash(), *usedGas-lastGas, receipt.Status)
		lastGas = *usedGas
		// END //
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	// Read requests if Prague is enabled.
	var requests [][]byte
	if p.config.IsPrague(block.Number(), block.Time()) {
		requests = [][]byte{}
		// EIP-6110
		if err := core.ParseDepositLogs(&requests, allLogs, p.config); err != nil {
			return nil, nil, 0, err
		}
		// EIP-7002
		core.ProcessWithdrawalQueue(&requests, evm)
		// EIP-7251
		core.ProcessConsolidationQueue(&requests, evm)
	}

	// Fail if Shanghai not enabled and len(withdrawals) is non-zero.
	withdrawals := block.Withdrawals()
	if len(withdrawals) > 0 && !p.config.IsShanghai(block.Number(), block.Time()) {
		return nil, nil, 0, errors.New("withdrawals before shanghai")
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	// START
	p.Engine.Finalize(p, header, worldState, block.Body())
	// END

	return receipts, allLogs, *usedGas, nil
}
