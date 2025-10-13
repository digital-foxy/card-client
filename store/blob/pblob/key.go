package pblob

import (
	"encoding/binary"
	"math"

	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/toolkit/timestamp"
)

const signBit = 0x8000000000000000

type dataType byte

const (
	minType     dataType = iota
	versionType dataType = iota
	thumbnailType

	maxType dataType = math.MaxUint8
)

type key struct {
	data []byte
	len  int
}

func newKey() key {
	return key{data: make([]byte, 17), len: 0}
}

func keyOf(b []byte) key {
	return key{data: b, len: len(b)}
}

func (k key) RID(rid resource.RID) key {
	binary.BigEndian.PutUint64(k.data[0:8], uint64(rid))
	if k.len < 8 {
		k.len = 8
	}
	return k
}

func (k key) Type(t dataType) key {
	k.data[8] = byte(t)
	if k.len < 9 {
		k.len = 9
	}
	return k
}

func (k key) Version(v timestamp.Nano) key {
	binary.BigEndian.PutUint64(k.data[9:17], uint64(v)^signBit)
	if k.len < 17 {
		k.len = 17
	}
	return k
}

func (k key) Bytes() []byte {
	return k.data[:k.len]
}

func (k key) GetRID() resource.RID {
	return resource.RID(binary.BigEndian.Uint64(k.data[0:]))
}

func (k key) GetType() dataType {
	return dataType(k.data[8])
}

func (k key) GetVersion() timestamp.Nano {
	return timestamp.Nano(binary.BigEndian.Uint64(k.data[9:]) ^ signBit)
}

func versionKey(rid resource.RID, version timestamp.Nano) []byte {
	return newKey().RID(rid).Type(versionType).Version(version).Bytes()
}

func versionPrefix(rid resource.RID) []byte {
	return newKey().RID(rid).Type(versionType).Bytes()
}

func thumbnailKey(rid resource.RID) []byte {
	return newKey().RID(rid).Type(thumbnailType).Bytes()
}
