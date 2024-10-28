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
	"github.com/ethereum/go-ethereum/common"
	"github.com/mus-format/mus-go/ord"
	"github.com/mus-format/mus-go/varint"
	itypes "github.com/wcgcyx/teler/types"
)

// marshalAccountValue implements the mus.Marshaller interface.
func marshalAccountValue(v *itypes.AccountValue, bs []byte) (n int) {
	n = varint.MarshalUint64(v.Nonce, bs)
	n += itypes.MarshalUint256(v.Balance, bs[n:])
	n += itypes.MarshalHash(v.CodeHash, bs[n:])
	n += ord.MarshalBool(v.DirtyStorage, bs[n:])
	n += varint.MarshalUint64(v.Version, bs[n:])
	return
}

// unmarshalAccountValue implements the mus.Unmarshaller interface.
func unmarshalAccountValue(bs []byte) (v *itypes.AccountValue, n int, err error) {
	v = &itypes.AccountValue{}
	v.Nonce, n, err = varint.UnmarshalUint64(bs)
	if err != nil {
		return
	}
	var n1 int
	v.Balance, n1, err = itypes.UnmarshalUint256(bs[n:])
	n += n1
	if err != nil {
		return
	}
	v.CodeHash, n1, err = itypes.UnmarshalHash(bs[n:])
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

// sizeAccountValue implements the mus.Sizer interface.
func sizeAccountValue(v *itypes.AccountValue) (size int) {
	size = varint.SizeUint64(v.Nonce)
	size += itypes.SizeUint256(v.Balance)
	size += itypes.SizeHash(v.CodeHash)
	size += ord.SizeBool(v.DirtyStorage)
	size += varint.SizeUint64(v.Version)
	return size
}

// rawCodeValue is used to track the code and its count seen by ds.
type rawCodeValue struct {
	code  []byte
	count int64
}

// marshalRawCodeValue implements the mus.Marshaller interface.
func marshalRawCodeValue(v rawCodeValue, bs []byte) (n int) {
	n = itypes.MarshalBytes(v.code, bs)
	n += varint.MarshalInt64(v.count, bs[n:])
	return
}

// unmarshalRawCodeValue implements the mus.Unmarshaller interface.
func unmarshalRawCodeValue(bs []byte) (v rawCodeValue, n int, err error) {
	v.code, n, err = itypes.UnmarshalBytes(bs)
	if err != nil {
		return
	}
	var n1 int
	v.count, n1, err = varint.UnmarshalInt64(bs[n:])
	n += n1
	return
}

// sizeRawCodeValue implements the mus.Sizer interface.
func sizeRawCodeValue(v rawCodeValue) (size int) {
	size = itypes.SizeBytes(v.code)
	size += varint.SizeInt64(v.count)
	return size
}

// persistedHeight is used to store height and root hash.
type persistedHeight struct {
	height uint64
	root   common.Hash
}

// marshalPersistedHeight implements the mus.Marshaller interface.
func marshalPersistedHeight(v persistedHeight, bs []byte) (n int) {
	n = varint.MarshalUint64(v.height, bs)
	n += itypes.MarshalHash(v.root, bs[n:])
	return
}

// unmarshalPersistedHeight implements the mus.Unmarshaller interface.
func unmarshalPersistedHeight(bs []byte) (v persistedHeight, n int, err error) {
	v.height, n, err = varint.UnmarshalUint64(bs)
	if err != nil {
		return
	}
	var n1 int
	v.root, n1, err = itypes.UnmarshalHash(bs[n:])
	n += n1
	return
}

// sizePersistedHeight implements the mus.Sizer interface.
func sizePersistedHeight(v persistedHeight) (size int) {
	size = varint.SizeUint64(v.height)
	size += itypes.SizeHash(v.root)
	return
}
