package rpc

/*
 * Licensed under LGPL-3.0.
 *
 * You can get a copy of the LGPL-3.0 License at
 *
 * https://www.gnu.org/licenses/lgpl-3.0.en.html
 *
 * @wcgcyx - https://github.com/wcgcyx
 */

import "time"

// Opts is the options for rpc.
type Opts struct {
	// The RPC Host
	Host string

	// The RPC Port
	Port uint64

	// The RPC Gas cap
	RPCGasCap uint64

	// The RPC EVM Timeout
	RPCEVMTimeout time.Duration
}
