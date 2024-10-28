package types

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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestAccountValueRoundTrip(t *testing.T) {
	original := AccountValue{
		11,
		uint256.NewInt(22),
		common.HexToHash("0xf48be2fbf5a8e6b02b456703b044fe0f3d3bdb45f6bd317c42278955edb27b55"),
		true,
		1,
	}

	size := SizeAccountValue(original)
	bs := make([]byte, size)
	MarshalAccountValue(original, bs)

	decoded, _, err := UnmarshalAccountValue(bs)
	assert.Nil(t, err)

	assert.Equal(t, original.Nonce, decoded.Nonce)
	assert.Equal(t, original.Balance, decoded.Balance)
	assert.Equal(t, original.CodeHash, decoded.CodeHash)
	assert.Equal(t, original.DirtyStorage, decoded.DirtyStorage)
	assert.Equal(t, original.Version, decoded.Version)
}

func TestLayerLogRoundTrip(t *testing.T) {
	original := LayerLog{
		BlockNumber: 123456,
		RootHash:    common.HexToHash("0xf48be2fbf5a8e6b02b456703b044fe0f3d3bdb45f6bd317c42278955edb27b55"),
		UpdatedCodeCount: map[common.Hash]int64{
			common.HexToHash("0xf48be2fbf5a8e6b02b456703b044fe0f3d3bdb45f6bd317c42278955edb27b55"): 1,
			common.HexToHash("0xf48be2fbf5a8e6b02b456703b044fe0f3d3bdb45f6bd317c42278955edb27b56"): 2,
			common.HexToHash("0xf48be2fbf5a8e6b02b456703b044fe0f3d3bdb45f6bd317c42278955edb27b57"): -1,
		},
		CodePreimage: map[common.Hash][]byte{
			common.HexToHash("0xf48be2fbf5a8e6b02b456703b044fe0f3d3bdb45f6bd317c42278955edb27b55"): {},
			common.HexToHash("0xf48be2fbf5a8e6b02b456703b044fe0f3d3bdb45f6bd317c42278955edb27b56"): {1, 2, 3, 4, 5},
		},
		UpdatedStorage: map[string]map[common.Hash]common.Hash{
			"0x976EA74026E726554dB657fA54763abd0C3a0aa9-1": {
				common.HexToHash("7"): common.HexToHash("0"),
			},
			"0x14dC79964da2C08b23698B3D3cc7Ca32193d9955-3": {
				common.HexToHash("8"): common.HexToHash("b"),
				common.HexToHash("9"): common.HexToHash("c"),
				common.HexToHash("a"): common.HexToHash("d"),
			},
		},
	}

	size := SizeLayerLog(original)
	bs := make([]byte, size)
	MarshalLayerLog(original, bs)

	decoded, _, err := UnmarshalLayerLog(bs)
	assert.Nil(t, err)

	assert.Equal(t, original.BlockNumber, decoded.BlockNumber)
	assert.Equal(t, original.RootHash, decoded.RootHash)
	assert.Equal(t, len(original.UpdatedAccounts), len(decoded.UpdatedAccounts))
	for addr, acct := range original.UpdatedAccounts {
		acct2 := decoded.UpdatedAccounts[addr]
		assert.Equal(t, acct.Nonce, acct2.Nonce)
		assert.Equal(t, acct.Balance, acct2.Balance)
		assert.Equal(t, acct.CodeHash, acct2.CodeHash)
		assert.Equal(t, acct.DirtyStorage, acct2.DirtyStorage)
		assert.Equal(t, acct.Version, acct2.Version)
	}
	assert.Equal(t, len(original.UpdatedCodeCount), len(decoded.UpdatedCodeCount))
	for codeHash, count := range original.UpdatedCodeCount {
		assert.Equal(t, count, decoded.UpdatedCodeCount[codeHash])
	}
	assert.Equal(t, len(original.CodePreimage), len(decoded.CodePreimage))
	for codeHash, code := range original.CodePreimage {
		code2 := decoded.CodePreimage[codeHash]
		assert.Equal(t, code, code2)
	}
	assert.Equal(t, len(original.UpdatedStorage), len(decoded.UpdatedStorage))
	for addr, storageMap := range original.UpdatedStorage {
		storageMap2 := decoded.UpdatedStorage[addr]
		assert.Equal(t, len(storageMap), len(storageMap2))
		for k, v := range storageMap {
			v2 := storageMap2[k]
			assert.Equal(t, v, v2)
		}
	}
}

func TestFromGenesis(t *testing.T) {
	mainnet := core.DefaultGenesisBlock()
	layerLog := LayerLogFromGenesis(mainnet, mainnet.ToBlock().Root())
	assert.Equal(t, uint64(0), layerLog.BlockNumber)
	assert.Equal(t, mainnet.ToBlock().Root(), layerLog.RootHash)
	assert.Equal(t, len(mainnet.Alloc), len(layerLog.UpdatedAccounts))
	assert.Equal(t, 0, len(layerLog.UpdatedCodeCount))
	assert.Equal(t, 0, len(layerLog.CodePreimage))
	assert.Equal(t, 0, len(layerLog.UpdatedStorage))
}
