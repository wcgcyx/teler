package statestore

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
	"encoding/base64"
	"encoding/binary"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ipfs/go-datastore"
	"github.com/mus-format/mus-go"
	"github.com/mus-format/mus-go/ord"
	itypes "github.com/wcgcyx/teler/types"
)

const (
	persistedKey      = "p"
	childrenKey       = "d"
	layerLogKey       = "l"
	accountValueKey   = "a"
	accountVersionKey = "v"
	storageKey        = "s"
	codeKey           = "c"
	gcKey             = "g"
	separator         = "/"
)

// persistedHeightKey gets the datastore key for persisted height.
func persistedHeightKey() datastore.Key {
	return datastore.NewKey(persistedKey)
}

// getChildrenKey gets the datastore key for children with given root.
func getChildrenKey(height uint64, root common.Hash) datastore.Key {
	rootStr := base64.URLEncoding.EncodeToString(root.Bytes())
	return datastore.NewKey(childrenKey + separator + strconv.FormatUint(height, 10) + separator + rootStr)
}

// getLayerLogKey gets the datastore key for layer log with given root.
func getLayerLogKey(height uint64, root common.Hash) datastore.Key {
	rootStr := base64.URLEncoding.EncodeToString(root.Bytes())
	return datastore.NewKey(layerLogKey + separator + strconv.FormatUint(height, 10) + separator + rootStr)
}

// getAccountValueKey gets the datastore key for account value with given address.
func getAccountValueKey(addr common.Address) datastore.Key {
	addrStr := base64.URLEncoding.EncodeToString(addr.Bytes())
	return datastore.NewKey(accountValueKey + separator + addrStr)
}

// getAccountVersionKey gets the datastore key for given storage version.
func getAccountVersionKey(addr common.Address) datastore.Key {
	addrStr := base64.URLEncoding.EncodeToString(addr.Bytes())
	return datastore.NewKey(accountVersionKey + separator + addrStr)
}

// getStorageKey gets the datastore key for given storage location.
func getStorageKey(addr common.Address, version uint64, key common.Hash) datastore.Key {
	addrStr := base64.URLEncoding.EncodeToString(addr.Bytes())
	keyStr := base64.URLEncoding.EncodeToString(key.Bytes())
	return datastore.NewKey(storageKey + separator + addrStr + separator + strconv.FormatUint(version, 10) + separator + keyStr)
}

// splitStorageKey splits the storage key to get location.
// This is unsafe, must be called on valid key string.
func splitStorageKey(key string) (common.Address, uint64, common.Hash) {
	temp := strings.Split(key, separator)
	data, _ := base64.URLEncoding.DecodeString(temp[1])
	addr := common.BytesToAddress(data)
	version, _ := strconv.ParseUint(temp[2], 10, 64)
	data, _ = base64.URLEncoding.DecodeString(temp[3])
	k := common.BytesToHash(data)
	return addr, version, k
}

// getCodeKey gets the datastore key for given address.
func getCodeKey(codeHash common.Hash) datastore.Key {
	codeStr := base64.URLEncoding.EncodeToString(codeHash.Bytes())
	return datastore.NewKey(codeKey + separator + codeStr)
}

// getGCKey gets the gc key for given address-version pair.
func getGCKey(addr common.Address, version uint64) datastore.Key {
	addrStr := base64.URLEncoding.EncodeToString(addr.Bytes())
	return datastore.NewKey(gcKey + separator + addrStr + separator + strconv.FormatUint(version, 10))
}

// splitGCKey splits the gc key to get address-version pair.
// This is unsafe, must be called on valid key string.
func splitGCKey(key string) (common.Address, uint64) {
	temp := strings.Split(key, separator)
	data, _ := base64.URLEncoding.DecodeString(temp[1])
	addr := common.BytesToAddress(data)
	version, _ := strconv.ParseUint(temp[2], 10, 64)
	return addr, version
}

// encodePersistedHeight encodes the persisted height and root.
func encodePersistedHeight(height uint64, root common.Hash) []byte {
	v := persistedHeight{
		height: height,
		root:   root,
	}
	size := sizePersistedHeight(v)
	bs := make([]byte, size)
	marshalPersistedHeight(v, bs)
	return bs
}

// decodePersistedHeight decodes the persisted height and root.
func decodePersistedHeight(bs []byte) (uint64, common.Hash, error) {
	v := persistedHeight{}
	v, _, err := unmarshalPersistedHeight(bs)
	return v.height, v.root, err
}

// encodeCodeValue encodes the code and count.
func encodeCodeValue(code []byte, count int64) []byte {
	v := rawCodeValue{
		code:  code,
		count: count,
	}
	size := sizeRawCodeValue(v)
	bs := make([]byte, size)
	marshalRawCodeValue(v, bs)
	return bs
}

// decodeCodeValue decodes the code and count.
func decodeCodeValue(bs []byte) ([]byte, int64, error) {
	v := rawCodeValue{}
	v, _, err := unmarshalRawCodeValue(bs)
	return v.code, v.count, err
}

// encodeChildren encodes the children.
func encodeChildren(children []common.Hash) []byte {
	s := mus.SizerFn[common.Hash](itypes.SizeHash)
	size := ord.SizeSlice[common.Hash](children, s)
	bs := make([]byte, size)
	m := mus.MarshallerFn[common.Hash](itypes.MarshalHash)
	ord.MarshalSlice[common.Hash](children, m, bs)
	return bs
}

// decodeChildren decodes the children.
func decodeChildren(val []byte) ([]common.Hash, error) {
	u := mus.UnmarshallerFn[common.Hash](itypes.UnmarshalHash)
	res, _, err := ord.UnmarshalSlice[common.Hash](u, val)
	return res, err
}

// encodeLayerLog encodes the layer log.
func encodeLayerLog(layerLog itypes.LayerLog) []byte {
	size := itypes.SizeLayerLog(layerLog)
	bs := make([]byte, size)
	itypes.MarshalLayerLog(layerLog, bs)
	return bs
}

// decodeLayerLog decodes the layer log.
func decodeLayerLog(val []byte) (itypes.LayerLog, error) {
	res, _, err := itypes.UnmarshalLayerLog(val)
	return res, err
}

// encodeAccountValue encodes the account value.
func encodeAccountValue(acct *itypes.AccountValue) []byte {
	size := sizeAccountValue(acct)
	bs := make([]byte, size)
	marshalAccountValue(acct, bs)
	return bs
}

// decodeAccountValue decodes the qaccount value.
func decodeAccountValue(val []byte) (*itypes.AccountValue, error) {
	res, _, err := unmarshalAccountValue(val)
	return res, err
}

// encodeAccountVersion encodes the account version.
func encodeAccountVersion(version uint64) []byte {
	versionBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(versionBytes, version)
	return versionBytes
}

// decodeAccountVersion decodes the account version.
func decodeAccountVersion(val []byte) uint64 {
	return binary.LittleEndian.Uint64(val)
}

// encodeStorage encodes the storage value.
func encodeStorage(val common.Hash) []byte {
	size := itypes.SizeHash(val)
	bs := make([]byte, size)
	itypes.MarshalHash(val, bs)
	return bs
}

// decodeStorage decodes the storage value.
func decodeStorage(val []byte) (common.Hash, error) {
	res, _, err := itypes.UnmarshalHash(val)
	return res, err
}
