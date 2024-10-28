package worldstate

/*
 * Licensed under LGPL-3.0.
 *
 * You can get a copy of the LGPL-3.0 License at
 *
 * https://www.gnu.org/licenses/lgpl-3.0.en.html
 *
 * @wcgcyx - https://github.com/wcgcyx
 */

// Opts is the options for world state.
type Opts struct {
	// Max layer to retain in memory
	MaxLayerToRetain uint64

	// The pruning frequency
	PruningFrequency uint64
}
