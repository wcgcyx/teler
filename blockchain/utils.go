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

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipfs/go-datastore"
	"github.com/mus-format/mus-go"
	"github.com/mus-format/mus-go/ord"
	"github.com/mus-format/mus-go/varint"
	itypes "github.com/wcgcyx/teler/types"
)

const (
	blockKey       = "block"
	transactionKey = "txn"
	receiptKey     = "receipt"
	forkKey        = "fork"
	tailKey        = "tail"
	headKey        = "head"
	finalizedKey   = "finalized"
	safeKey        = "safe"
	separator      = "/"
)

// getBlockKey gets the datastore key for given block hash.
func getBlockKey(hash common.Hash) datastore.Key {
	return datastore.NewKey(blockKey + separator + hash.Hex())
}

// getTransactionKey gets the datastore key for given txn hash.
func getTransactionKey(hash common.Hash) datastore.Key {
	return datastore.NewKey(transactionKey + separator + hash.Hex())
}

// getReceiptKey gets the datastore key for receipt of given txn hash.
func getReceiptKey(hash common.Hash) datastore.Key {
	return datastore.NewKey(receiptKey + separator + hash.Hex())
}

// getForkKey gets the datastore key for given fork number.
func getForkKey(height uint64) datastore.Key {
	return datastore.NewKey(forkKey + separator + strconv.FormatUint(height, 10))
}

// getTailKey gets the datastore key for tail block.
func getTailKey() datastore.Key {
	return datastore.NewKey(tailKey)
}

// getHeadKey gets the datastore key for head block.
func getHeadKey() datastore.Key {
	return datastore.NewKey(headKey)
}

// getFinalizedKey gets the datastore key for finalized block.
func getFinalizedKey() datastore.Key {
	return datastore.NewKey(finalizedKey)
}

// getSafeKey gets the datastore key for safe block.
func getSafeKey() datastore.Key {
	return datastore.NewKey(safeKey)
}

// encodeBlock encodes block to bytes.
func encodeBlock(v *types.Block) []byte {
	buf := new(bytes.Buffer)
	v.EncodeRLP(buf)
	return buf.Bytes()
}

// decodeBlock decodes bytes to block.
func decodeBlock(val []byte) (*types.Block, error) {
	res := &types.Block{}
	err := res.DecodeRLP(rlp.NewStream(bytes.NewReader(val), 0))
	if err != nil {
		return nil, err
	}
	return res, nil
}

// encodeForks encodes forks to bytes.
func encodeForks(v []common.Hash) []byte {
	s := mus.SizerFn[common.Hash](itypes.SizeHash)
	size := ord.SizeSlice[common.Hash](v, s)
	bs := make([]byte, size)
	m := mus.MarshallerFn[common.Hash](itypes.MarshalHash)
	ord.MarshalSlice[common.Hash](v, m, bs)
	return bs
}

// decodeBlock decodes bytes to forks.
func decodeForks(val []byte) ([]common.Hash, error) {
	u := mus.UnmarshallerFn[common.Hash](itypes.UnmarshalHash)
	res, _, err := ord.UnmarshalSlice[common.Hash](u, val)
	return res, err
}

// encodeTransaction encodes transaction to bytes.
func encodeTransaction(v *types.Transaction, blkHash common.Hash, index uint64) []byte {
	buf := new(bytes.Buffer)
	v.EncodeRLP(buf)
	txData := buf.Bytes()

	size := itypes.SizeBytes(txData)
	size += itypes.SizeHash(blkHash)
	size += varint.SizeUint64(index)
	bs := make([]byte, size)

	n := itypes.MarshalBytes(txData, bs)
	n += itypes.MarshalHash(blkHash, bs[n:])
	varint.MarshalUint64(index, bs[n:])

	return bs
}

// decodeTransaction decodes bytes to transaction.
func decodeTransaction(val []byte) (*types.Transaction, common.Hash, uint64, error) {
	txData, n, err := itypes.UnmarshalBytes(val)
	if err != nil {
		return nil, common.Hash{}, 0, err
	}
	res := &types.Transaction{}
	err = res.DecodeRLP(rlp.NewStream(bytes.NewReader(txData), 0))
	if err != nil {
		return nil, common.Hash{}, 0, err
	}
	blkHash, n1, err := itypes.UnmarshalHash(val[n:])
	n += n1
	if err != nil {
		return nil, common.Hash{}, 0, err
	}
	index, _, err := varint.UnmarshalUint64(val[n:])
	if err != nil {
		return nil, common.Hash{}, 0, err
	}
	return res, blkHash, index, nil
}

// encodeReceipt encodes receipt to bytes.
func encodeReceipt(v *types.Receipt, blkHash common.Hash, index uint64) []byte {
	receiptData, _ := json.Marshal(v)
	size := itypes.SizeBytes(receiptData)
	size += itypes.SizeHash(blkHash)
	size += varint.SizeUint64(index)
	bs := make([]byte, size)

	n := itypes.MarshalBytes(receiptData, bs)
	n += itypes.MarshalHash(blkHash, bs[n:])
	varint.MarshalUint64(index, bs[n:])

	return bs
}

// decodeReceipt decodes bytes to receipt.
func decodeReceipt(val []byte) (*types.Receipt, common.Hash, uint64, error) {
	receiptData, n, err := itypes.UnmarshalBytes(val)
	if err != nil {
		return nil, common.Hash{}, 0, err
	}
	res := &types.Receipt{}
	err = json.Unmarshal(receiptData, res)
	if err != nil {
		return nil, common.Hash{}, 0, err
	}
	blkHash, n1, err := itypes.UnmarshalHash(val[n:])
	n += n1
	if err != nil {
		return nil, common.Hash{}, 0, err
	}
	index, _, err := varint.UnmarshalUint64(val[n:])
	if err != nil {
		return nil, common.Hash{}, 0, err
	}
	return res, blkHash, index, nil
}
