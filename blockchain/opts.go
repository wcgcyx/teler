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

import "time"

// Opts is the options for blockchain.
type Opts struct {
	// Path to the data store
	Path string

	// Max block to retain
	MaxBlockToRetain uint64

	// The pruning frequency
	PruningFrequency uint64

	// The IO Timeout
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}
