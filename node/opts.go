package node

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

// Opts is the options for node.
type Opts struct {
	// The frequency at which node checks for new block.
	CheckFrequency time.Duration

	// The max allowed blocks local can lead remote.
	MaxAllowedLeadBlocks uint64

	// The minimum distance from local to remote to start forward sync.
	ForwardSyncMinDistanceToStart uint64

	// The gap between remote head and the forward sync target.
	ForwardSyncTargetGap uint64

	// The max allowed blocks to query in backward sync.
	BackwardSyncMaxBlocksToQuery uint64

	// The max allowed blocks to query in backward sync (during reorg).
	BackwardSyncMaxBlocksToQueryReorg uint64
}
