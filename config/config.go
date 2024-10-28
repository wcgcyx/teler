package config

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
	"os"
	"path/filepath"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/spf13/viper"
)

// Logger
var log = logging.Logger("config")

const (
	defaultConfigPath = ".teler"
)

type Config struct {
	// Global
	GlobalLoggingLevel string        `mapstructure:"LOGGING"`          // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	Path               string        `mapstructure:"DATA_DIR"`         // Main datastore path.
	BlockSourceURL     string        `mapstructure:"BLOCK_SOURCE_URL"` // Main datastore path.
	DSTimeout          time.Duration `mapstructure:"DS_TIMEOUT"`       // Datastore timeout.

	// Statestore
	StateStoreGCPeriod time.Duration `mapstructure:"STATESTORE_GC_PERIOD"` // Statestore GC period.

	// Blockchain
	BlockchainMaxBlockToRetain uint64 `mapstructure:"BLOCKCHAIN_MAX_BLOCK_TO_RETAIN"` // Blockchain max block to retain.
	BlockchainPruningFrequency uint64 `mapstructure:"BLOCKCHAIN_PRUNING_FREQUENCY"`   // Blockchain pruning frequency.

	// WorldState
	WorldStateMaxLayerToRetain uint64 `mapstructure:"WORLDSTATE_MAX_LAYER_TO_RETAIN"` // World state max layer to retain.
	WorldStatePruningFrequency uint64 `mapstructure:"WORLDSTATE_PRUNING_FREQUENCY"`   // World state pruning frequency.

	// RPC
	RPCHost       string        `mapstructure:"RPC_HOST"`        // RPC Server host.
	RPCPort       uint64        `mapstructure:"RPC_PORT"`        // RPC Server port.
	RPCGasCap     uint64        `mapstructure:"RPC_GAS_CAP"`     // RPC gas cap for answering calls.
	RPCEVMTimeout time.Duration `mapstructure:"RPC_EVM_TIMEOUT"` // RPC evm timeout for answering calls.

	// Node
	NodeCheckFrequency                    time.Duration `mapstructure:"NODE_CHECK_FREQUENCY"`                // Frequency node checks for new blocks.
	NodeMaxAllowedLeadBlocks              uint64        `mapstructure:"NODE_MAX_ALLOWED_LEAD_BLOCKS"`        // The max allowed blocks local can lead remote.
	NodeForwardSyncMinDistanceToStart     uint64        `mapstructure:"NODE_FORWARD_SYNC_MIN_DIST"`          // The minimum distance from local to remote to start forward sync.
	NodeForwardSyncTargetGap              uint64        `mapstructure:"NODE_FORWARD_SYNC_TARGET_GAP"`        // The gap between remote head and the forward sync target.
	NodeBackwardSyncMaxBlocksToQuery      uint64        `mapstructure:"NODE_BACKWARD_SYNC_MAX_BLOCKS"`       // The max allowed blocks to query in backward sync.
	NodeBackwardSyncMaxBlocksToQueryReorg uint64        `mapstructure:"NODE_BACKWARD_SYNC_MAX_BLOCKS_REORG"` // The max allowed blocks to query in backward sync  (during reorg).
}

// Default configs
var DefaultConfig Config = Config{
	Path:                                  "$HOME/.teler",
	BlockSourceURL:                        "http://localhost:8545",
	GlobalLoggingLevel:                    "INFO",
	DSTimeout:                             5 * time.Second,
	StateStoreGCPeriod:                    30 * time.Minute,
	BlockchainMaxBlockToRetain:            512,
	BlockchainPruningFrequency:            257,
	WorldStateMaxLayerToRetain:            256,
	WorldStatePruningFrequency:            127,
	RPCHost:                               "localhost",
	RPCPort:                               9424,
	RPCGasCap:                             50000000,
	RPCEVMTimeout:                         5 * time.Second,
	NodeCheckFrequency:                    10 * time.Second,
	NodeMaxAllowedLeadBlocks:              64,
	NodeForwardSyncMinDistanceToStart:     256,
	NodeForwardSyncTargetGap:              64,
	NodeBackwardSyncMaxBlocksToQuery:      260,
	NodeBackwardSyncMaxBlocksToQueryReorg: 64,
}

// NewConfig creates a new configuration.
//
// @output - configuration, error.
func NewConfig(configFile string) (Config, error) {
	// Try to load config file from $HOME/.teler
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/" + defaultConfigPath)
	if configFile != "" {
		viper.SetConfigFile(configFile)
	}
	viper.AutomaticEnv()

	conf := Config{}

	// Parse global config
	conf.GlobalLoggingLevel = viper.GetString("LOGGING")
	if conf.GlobalLoggingLevel == "" {
		conf.GlobalLoggingLevel = DefaultConfig.GlobalLoggingLevel
	}
	logLevel, err := logging.LevelFromString(conf.GlobalLoggingLevel)
	if err != nil {
		return Config{}, err
	}
	logging.SetAllLoggers(logLevel)
	conf.Path = viper.GetString("DATA_DIR")
	if conf.Path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, err
		}
		conf.Path = filepath.Join(home, DefaultConfig.Path)
		log.Infof("DATA_DIR not defined, use default: %v", conf.Path)
	}
	conf.BlockSourceURL = viper.GetString("BLOCK_SOURCE_URL")
	if conf.BlockSourceURL == "" {
		conf.BlockSourceURL = DefaultConfig.BlockSourceURL
		log.Infof("BLOCK_SOURCE_URL not defined, use default: %v", conf.BlockSourceURL)
	}
	conf.DSTimeout = viper.GetDuration("DS_TIMEOUT")
	if conf.DSTimeout <= 0 {
		conf.DSTimeout = DefaultConfig.DSTimeout
		log.Infof("Invalid DS_TIMEOUT found, use default: %v", conf.DSTimeout)
	}

	// Parse statestore config
	conf.StateStoreGCPeriod = viper.GetDuration("STATESTORE_GC_PERIOD")
	if conf.StateStoreGCPeriod < 10*time.Minute {
		conf.StateStoreGCPeriod = DefaultConfig.StateStoreGCPeriod
		log.Infof("STATESTORE_GC_PERIOD is smaller than min 10m, use default %v", conf.StateStoreGCPeriod)
	}

	// Parse blockchain config
	conf.BlockchainMaxBlockToRetain = uint64(viper.GetInt64("BLOCKCHAIN_MAX_BLOCK_TO_RETAIN"))
	if conf.BlockchainMaxBlockToRetain < 128 || conf.BlockchainMaxBlockToRetain > 7200 {
		conf.BlockchainMaxBlockToRetain = DefaultConfig.BlockchainMaxBlockToRetain
		log.Infof("BLOCKCHAIN_MAX_BLOCK_TO_RETAIN is not between 128 and 7200, use default: %v", conf.BlockchainMaxBlockToRetain)
	}
	conf.BlockchainPruningFrequency = uint64(viper.GetInt64("BLOCKCHAIN_PRUNING_FREQUENCY"))
	if conf.BlockchainPruningFrequency < 64 || conf.BlockchainPruningFrequency > conf.BlockchainMaxBlockToRetain {
		conf.BlockchainPruningFrequency = DefaultConfig.BlockchainPruningFrequency
		log.Infof("BLOCKCHAIN_PRUNING_FREQUENCY is not between 64 and BLOCKCHAIN_MAX_BLOCK_TO_RETAIN, use default: %v", conf.BlockchainPruningFrequency)
	}

	// Parse worldstate config
	conf.WorldStateMaxLayerToRetain = uint64(viper.GetInt64("WORLDSTATE_MAX_LAYER_TO_RETAIN"))
	if conf.WorldStateMaxLayerToRetain < 128 || conf.WorldStateMaxLayerToRetain > 512 {
		conf.WorldStateMaxLayerToRetain = DefaultConfig.WorldStateMaxLayerToRetain
		log.Infof("WORLDSTATE_MAX_LAYER_TO_RETAIN is not between 128 and 512, use default: %v", conf.WorldStateMaxLayerToRetain)
	}
	conf.WorldStatePruningFrequency = uint64(viper.GetInt64("WORLDSTATE_PRUNING_FREQUENCY"))
	if conf.WorldStatePruningFrequency < 64 || conf.WorldStatePruningFrequency > conf.WorldStateMaxLayerToRetain {
		conf.WorldStatePruningFrequency = DefaultConfig.WorldStatePruningFrequency
		log.Infof("WORLDSTATE_PRUNING_FREQUENCY is not between 64 and WORLDSTATE_MAX_LAYER_TO_RETAIN, use default: %v", conf.WorldStatePruningFrequency)
	}

	// Parse RPC config
	conf.RPCHost = viper.GetString("RPC_HOST")
	if conf.RPCHost == "" {
		conf.RPCHost = DefaultConfig.RPCHost
		log.Infof("RPC_HOST not set, use default %v", conf.RPCHost)
	}
	conf.RPCPort = uint64(viper.GetInt64("RPC_PORT"))
	if conf.RPCPort == 0 {
		conf.RPCPort = DefaultConfig.RPCPort
		log.Infof("RPC_PORT not set, use default %v", conf.RPCPort)
	}
	conf.RPCGasCap = uint64(viper.GetInt64("RPC_GAS_CAP"))
	if conf.RPCGasCap == 0 {
		conf.RPCGasCap = DefaultConfig.RPCGasCap
		log.Infof("RPC_GAS_CAP not set, use default %v", conf.RPCGasCap)
	}
	conf.RPCEVMTimeout = viper.GetDuration("RPC_EVM_TIMEOUT")
	if conf.RPCEVMTimeout <= 0 {
		conf.RPCEVMTimeout = DefaultConfig.RPCEVMTimeout
		log.Infof("Invalid RPC_EVM_TIMEOUT found, use default: %v", conf.RPCEVMTimeout)
	}

	// TODO: Support the configuration of node syncing behaviour.
	conf.NodeCheckFrequency = DefaultConfig.NodeCheckFrequency
	conf.NodeMaxAllowedLeadBlocks = DefaultConfig.NodeMaxAllowedLeadBlocks
	conf.NodeForwardSyncMinDistanceToStart = DefaultConfig.NodeForwardSyncMinDistanceToStart
	conf.NodeForwardSyncTargetGap = DefaultConfig.NodeForwardSyncTargetGap
	conf.NodeBackwardSyncMaxBlocksToQuery = DefaultConfig.NodeBackwardSyncMaxBlocksToQuery
	conf.NodeBackwardSyncMaxBlocksToQueryReorg = DefaultConfig.NodeBackwardSyncMaxBlocksToQueryReorg

	return conf, nil
}
