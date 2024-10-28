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
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/mus-format/mus-go"
	"github.com/mus-format/mus-go/ord"
	"github.com/mus-format/mus-go/varint"
)

// GetAccountStorageKey gets the storage key for a given address and version.
func GetAccountStorageKey(addr common.Address, version uint64) string {
	return addr.Hex() + "-" + strconv.FormatUint(version, 10)
}

// SplitAccountStorageKey splits the storage key to get the address and version.
// This is unsafe, must be called on valid key string.
func SplitAccountStorageKey(key string) (common.Address, uint64) {
	temp := strings.Split(key, "-")
	addr := common.HexToAddress(temp[0])
	version, _ := strconv.ParseUint(temp[1], 10, 64)
	return addr, version
}

// MarshalAddress implements the mus.Marshaller interface.
func MarshalAddress(v common.Address, bs []byte) (n int) {
	sl := v.Bytes()
	m := mus.MarshallerFn[byte](varint.MarshalByte)
	n = ord.MarshalSlice[byte](sl, m, bs)
	return
}

// UnmarshalAddress implements the mus.Unmarshaller interface.
func UnmarshalAddress(bs []byte) (v common.Address, n int, err error) {
	var sl []byte
	u := mus.UnmarshallerFn[byte](varint.UnmarshalByte)
	sl, n, err = ord.UnmarshalSlice[byte](u, bs)
	if err != nil {
		return
	}
	v.SetBytes(sl)
	return
}

// SizeAddress implements the mus.Sizer interface.
func SizeAddress(v common.Address) (size int) {
	sl := v.Bytes()
	s := mus.SizerFn[byte](varint.SizeByte)
	size = ord.SizeSlice[byte](sl, s)
	return
}

// MarshalHash implements the mus.Marshaller interface.
func MarshalHash(v common.Hash, bs []byte) (n int) {
	sl := v.Bytes()
	m := mus.MarshallerFn[byte](varint.MarshalByte)
	n = ord.MarshalSlice[byte](sl, m, bs)
	return
}

// UnmarshalHash implements the mus.Unmarshaller interface.
func UnmarshalHash(bs []byte) (v common.Hash, n int, err error) {
	var sl []byte
	u := mus.UnmarshallerFn[byte](varint.UnmarshalByte)
	sl, n, err = ord.UnmarshalSlice[byte](u, bs)
	if err != nil {
		return
	}
	v.SetBytes(sl)
	return
}

// SizeHash implements the mus.Sizer interface.
func SizeHash(v common.Hash) (size int) {
	sl := v.Bytes()
	s := mus.SizerFn[byte](varint.SizeByte)
	size = ord.SizeSlice[byte](sl, s)
	return
}

// MarshalUint256 implements the mus.Marshaller interface.
func MarshalUint256(v *uint256.Int, bs []byte) (n int) {
	sl := v.Bytes()
	m := mus.MarshallerFn[byte](varint.MarshalByte)
	n = ord.MarshalSlice[byte](sl, m, bs)
	return
}

// UnmarshalUint256 implements the mus.Unmarshaller interface.
func UnmarshalUint256(bs []byte) (v *uint256.Int, n int, err error) {
	var sl []byte
	u := mus.UnmarshallerFn[byte](varint.UnmarshalByte)
	sl, n, err = ord.UnmarshalSlice[byte](u, bs)
	if err != nil {
		return
	}
	v = uint256.NewInt(0).SetBytes(sl)
	return
}

// SizeUint256 implements the mus.Sizer interface.
func SizeUint256(v *uint256.Int) (size int) {
	sl := v.Bytes()
	s := mus.SizerFn[byte](varint.SizeByte)
	size = ord.SizeSlice[byte](sl, s)
	return
}

// MarshalBytes implements the mus.Marshaller interface.
func MarshalBytes(v []byte, bs []byte) (n int) {
	m := mus.MarshallerFn[byte](varint.MarshalByte)
	n = ord.MarshalSlice[byte](v, m, bs)
	return
}

// UnmarshalBytes implements the mus.Unmarshaller interface.
func UnmarshalBytes(bs []byte) (v []byte, n int, err error) {
	u := mus.UnmarshallerFn[byte](varint.UnmarshalByte)
	v, n, err = ord.UnmarshalSlice[byte](u, bs)
	return
}

// SizeBytes implements the mus.Sizer interface.
func SizeBytes(v []byte) (size int) {
	s := mus.SizerFn[byte](varint.SizeByte)
	size = ord.SizeSlice[byte](v, s)
	return
}

// MarshalStorage implements the mus.Marshaller interface.
func MarshalStorage(v map[common.Hash]common.Hash, bs []byte) (n int) {
	m := mus.MarshallerFn[common.Hash](MarshalHash)
	n = ord.MarshalMap[common.Hash, common.Hash](v, m, m, bs)
	return
}

// UnmarshalStorage implements the mus.Unmarshaller interface.
func UnmarshalStorage(bs []byte) (v map[common.Hash]common.Hash, n int, err error) {
	u := mus.UnmarshallerFn[common.Hash](UnmarshalHash)
	v, n, err = ord.UnmarshalMap[common.Hash, common.Hash](u, u, bs)
	return
}

// SizeStorage implements the mus.Sizer interface.
func SizeStorage(v map[common.Hash]common.Hash) (size int) {
	s := mus.SizerFn[common.Hash](SizeHash)
	size = ord.SizeMap[common.Hash, common.Hash](v, s, s)
	return
}

// MarshalAccountValue implements the mus.Marshaller interface.
func MarshalAccountValue(v AccountValue, bs []byte) (n int) {
	n = varint.MarshalUint64(v.Nonce, bs)
	n += MarshalUint256(v.Balance, bs[n:])
	n += MarshalHash(v.CodeHash, bs[n:])
	n += ord.MarshalBool(v.DirtyStorage, bs[n:])
	n += varint.MarshalUint64(v.Version, bs[n:])
	return
}

// UnmarshalAccountValue implements the mus.Unmarshaller interface.
func UnmarshalAccountValue(bs []byte) (v AccountValue, n int, err error) {
	v = AccountValue{}
	v.Nonce, n, err = varint.UnmarshalUint64(bs)
	if err != nil {
		return
	}
	var n1 int
	v.Balance, n1, err = UnmarshalUint256(bs[n:])
	n += n1
	if err != nil {
		return
	}
	v.CodeHash, n1, err = UnmarshalHash(bs[n:])
	n += n1
	if err != nil {
		return
	}
	v.DirtyStorage, n1, err = ord.UnmarshalBool(bs[n:])
	n += n1
	if err != nil {
		return
	}
	v.Version, n1, err = varint.UnmarshalUint64(bs[n:])
	n += n1
	return
}

// SizeAccountValue implements the mus.Sizer interface.
func SizeAccountValue(v AccountValue) (size int) {
	size = varint.SizeUint64(v.Nonce)
	size += SizeUint256(v.Balance)
	size += SizeHash(v.CodeHash)
	size += ord.SizeBool(v.DirtyStorage)
	size += varint.SizeUint64(v.Version)
	return size
}

// MarshalLayerLog implements the mus.Marshaller interface.
func MarshalLayerLog(v LayerLog, bs []byte) (n int) {
	n = varint.MarshalUint64(v.BlockNumber, bs)
	n += MarshalHash(v.RootHash, bs[n:])
	m1 := mus.MarshallerFn[common.Address](MarshalAddress)
	m2 := mus.MarshallerFn[AccountValue](MarshalAccountValue)
	n += ord.MarshalMap[common.Address, AccountValue](v.UpdatedAccounts, m1, m2, bs[n:])
	m3 := mus.MarshallerFn[common.Hash](MarshalHash)
	m4 := mus.MarshallerFn[int64](varint.MarshalInt64)
	n += ord.MarshalMap[common.Hash, int64](v.UpdatedCodeCount, m3, m4, bs[n:])
	m5 := mus.MarshallerFn[[]byte](MarshalBytes)
	n += ord.MarshalMap[common.Hash, []byte](v.CodePreimage, m3, m5, bs[n:])
	m6 := mus.MarshallerFn[string](ord.MarshalString)
	m7 := mus.MarshallerFn[map[common.Hash]common.Hash](MarshalStorage)
	n += ord.MarshalMap[string, map[common.Hash]common.Hash](v.UpdatedStorage, m6, m7, bs[n:])
	return n
}

// UnmarshalLayerLog implements the mus.Unmarshaller interface.
func UnmarshalLayerLog(bs []byte) (v LayerLog, n int, err error) {
	v.BlockNumber, n, err = varint.UnmarshalUint64(bs)
	if err != nil {
		return
	}
	var n1 int
	v.RootHash, n1, err = UnmarshalHash(bs[n:])
	n += n1
	if err != nil {
		return
	}
	u1 := mus.UnmarshallerFn[common.Address](UnmarshalAddress)
	u2 := mus.UnmarshallerFn[AccountValue](UnmarshalAccountValue)
	v.UpdatedAccounts, n1, err = ord.UnmarshalMap[common.Address, AccountValue](u1, u2, bs[n:])
	n += n1
	if err != nil {
		return
	}
	u3 := mus.UnmarshallerFn[common.Hash](UnmarshalHash)
	u4 := mus.UnmarshallerFn[int64](varint.UnmarshalInt64)
	v.UpdatedCodeCount, n1, err = ord.UnmarshalMap[common.Hash, int64](u3, u4, bs[n:])
	n += n1
	if err != nil {
		return
	}
	u5 := mus.UnmarshallerFn[[]byte](UnmarshalBytes)
	v.CodePreimage, n1, err = ord.UnmarshalMap[common.Hash, []byte](u3, u5, bs[n:])
	n += n1
	if err != nil {
		return
	}
	u6 := mus.UnmarshallerFn[string](ord.UnmarshalString)
	u7 := mus.UnmarshallerFn[map[common.Hash]common.Hash](UnmarshalStorage)
	v.UpdatedStorage, n1, err = ord.UnmarshalMap[string, map[common.Hash]common.Hash](u6, u7, bs[n:])
	n += n1
	return
}

// SizeLayerLog implements the mus.Sizer interface.
func SizeLayerLog(v LayerLog) (size int) {
	size = varint.SizeUint64(v.BlockNumber)
	size += SizeHash(v.RootHash)
	s1 := mus.SizerFn[common.Address](SizeAddress)
	s2 := mus.SizerFn[AccountValue](SizeAccountValue)
	size += ord.SizeMap[common.Address, AccountValue](v.UpdatedAccounts, s1, s2)
	s3 := mus.SizerFn[common.Hash](SizeHash)
	s4 := mus.SizerFn[int64](varint.SizeInt64)
	size += ord.SizeMap[common.Hash, int64](v.UpdatedCodeCount, s3, s4)
	s5 := mus.SizerFn[[]byte](SizeBytes)
	size += ord.SizeMap[common.Hash, []byte](v.CodePreimage, s3, s5)
	s6 := mus.SizerFn[string](ord.SizeString)
	s7 := mus.SizerFn[map[common.Hash]common.Hash](SizeStorage)
	size += ord.SizeMap[string, map[common.Hash]common.Hash](v.UpdatedStorage, s6, s7)
	return size
}
