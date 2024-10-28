package cli

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
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient"
	logging "github.com/ipfs/go-log"
	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/teler/backend"
	"github.com/wcgcyx/teler/blockchain"
	"github.com/wcgcyx/teler/config"
	"github.com/wcgcyx/teler/node"
	"github.com/wcgcyx/teler/processor"
	"github.com/wcgcyx/teler/rpc"
	"github.com/wcgcyx/teler/statestore"
	"github.com/wcgcyx/teler/worldstate"
)

// Logger
var log = logging.Logger("node")

func runNode(c *cli.Context) error {
	// Load config
	conf, err := config.NewConfig(c.String("config"))
	if err != nil {
		return err
	}
	if c.IsSet("blksrc") {
		log.Infof("Override blksrc to be %v", c.String("blksrc"))
		conf.BlockSourceURL = c.String("blksrc")
	}
	if c.IsSet("path") {
		log.Infof("Override path to be %v", c.String("path"))
		conf.Path = c.String("path")
	}
	if c.IsSet("rpc-host") {
		log.Infof("Override rpc-host to be %v", c.String("rpc-host"))
		conf.RPCHost = c.String("rpc-host")
	}
	if c.IsSet("rpc-port") {
		log.Infof("Override rpc-port to be %v", c.String("rpc-port"))
		conf.RPCPort = uint64(c.Int("rpc-port"))
	}
	// Read genesis
	chain := "mainnet"
	if _, err := os.Stat(filepath.Join(conf.Path, "genesis")); errors.Is(err, os.ErrNotExist) {
		// Read from user
		if c.IsSet("chain") {
			chain = c.String("chain")
		}
	} else {
		// Read from local
		if c.IsSet("chain") {
			log.Infof("Ignored chain option: %v", c.String("chain"))
		}
		contents, err := os.ReadFile(filepath.Join(conf.Path, "genesis"))
		if err != nil {
			return err
		}
		chain = string(contents)
	}
	var genesis *core.Genesis
	if chain == "mainnet" {
		genesis = core.DefaultGenesisBlock()
	} else if chain == "sepolia" {
		genesis = core.DefaultSepoliaGenesisBlock()
	} else if chain == "holesky" {
		genesis = core.DefaultHoleskyGenesisBlock()
	} else {
		return fmt.Errorf("unsupported chain %v", chain)
	}
	// Write to local
	log.Infof("Use chain config for %v", chain)
	os.WriteFile(filepath.Join(conf.Path, "genesis"), []byte(chain), os.ModePerm)

	// Create blockchain
	log.Infof("Start blockchain...")
	bc, err := blockchain.NewBlockchainImpl(c.Context, blockchain.Opts{
		Path:             filepath.Join(conf.Path, "chaindata"),
		MaxBlockToRetain: conf.BlockchainMaxBlockToRetain,
		PruningFrequency: conf.BlockchainPruningFrequency,
		ReadTimeout:      conf.DSTimeout,
		WriteTimeout:     conf.DSTimeout,
	}, genesis)
	if err != nil {
		return err
	}
	log.Infof("Blockchain started.")
	defer bc.Shutdown()

	// Create remote block source
	blkSrc, err := ethclient.Dial(conf.BlockSourceURL)
	if err != nil {
		return err
	}
	defer blkSrc.Close()

	// Create statestore
	log.Infof("Start statestore...")
	sstore, err := statestore.NewStateStoreImpl(c.Context, statestore.Opts{
		Path:         filepath.Join(conf.Path, "statedata"),
		GCPeriod:     conf.StateStoreGCPeriod,
		ReadTimeout:  conf.DSTimeout,
		WriteTimeout: conf.DSTimeout,
	}, genesis, genesis.ToBlock().Root())
	if err != nil {
		return err
	}
	defer sstore.Shutdown()
	log.Infof("Statestore started.")

	// Create world state
	log.Infof("Start worldstate archive...")
	archive, err := worldstate.NewLayeredWorldStateArchiveImpl(worldstate.Opts{
		MaxLayerToRetain: conf.WorldStateMaxLayerToRetain,
		PruningFrequency: conf.WorldStatePruningFrequency,
	}, genesis.Config, sstore)
	if err != nil {
		return err
	}
	log.Infof("Worldstate archive started.")

	// Create block processor
	pr, err := processor.NewBlockProcessor(genesis.Config, bc)
	if err != nil {
		return err
	}

	// Create backend
	be := backend.NewBackendImpl(pr, genesis.Config, bc, archive)

	// Create node
	node, err := node.NewNode(node.Opts{
		CheckFrequency:                    conf.NodeCheckFrequency,
		MaxAllowedLeadBlocks:              conf.NodeMaxAllowedLeadBlocks,
		ForwardSyncMinDistanceToStart:     conf.NodeForwardSyncMinDistanceToStart,
		ForwardSyncTargetGap:              conf.NodeForwardSyncTargetGap,
		BackwardSyncMaxBlocksToQuery:      conf.NodeBackwardSyncMaxBlocksToQuery,
		BackwardSyncMaxBlocksToQueryReorg: conf.NodeBackwardSyncMaxBlocksToQueryReorg,
	}, blkSrc, be)
	if err != nil {
		return err
	}

	// Create API server
	log.Infof("Start API Server...")
	apiServer, err := rpc.NewServer(rpc.Opts{
		Host:          conf.RPCHost,
		Port:          conf.RPCPort,
		RPCGasCap:     conf.RPCGasCap,
		RPCEVMTimeout: conf.RPCEVMTimeout,
	}, node)
	if err != nil {
		return err
	}
	log.Infof("Start serving at %v:%v", conf.RPCHost, conf.RPCPort)

	// Start mainloop
	go node.Mainloop()

	// Configure graceful shutdown.
	cc := make(chan os.Signal, 1)
	signal.Notify(cc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	for {
		// Loop forever, until exit
		<-cc
		break
	}
	log.Infof("Graceful shutdown...")
	apiServer.Shutdown()
	node.Shutdown()
	return nil
}
