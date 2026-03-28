package pblob

import (
	"encoding/binary"
	"math"

	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/toolkit/timestamp"
)

const signBit = 0x8000000000000000

const (
	typePosition = 8
	maxKeyLength = 17
)

// dataType identifies the type of data stored (png, json, thumbnail)
type dataType byte

const (
	pngType       dataType = 0x00
	jsonType      dataType = 0x01
	thumbnailType dataType = 0x7F

	minType dataType = 0x00
	maxType dataType = math.MaxUint8
)

// key is a composite key for Pebble: RID + type + version
type key struct {
	data []byte
	len  int
}

func newKey() key {
	return key{data: make([]byte, maxKeyLength), len: 0}
}

func keyOf(b []byte) key {
	return key{data: b, len: len(b)}
}

func (k key) RID(rid resource.RID) key {
	binary.BigEndian.PutUint64(k.data[0:typePosition], uint64(rid))
	if k.len < typePosition {
		k.len = typePosition
	}
	return k
}

func (k key) Type(t dataType) key {
	k.data[typePosition] = byte(t)
	if k.len < typePosition+1 {
		k.len = typePosition + 1
	}
	return k
}

func (k key) Version(v timestamp.Nano) key {
	binary.BigEndian.PutUint64(k.data[typePosition+1:maxKeyLength], uint64(v)^signBit)
	if k.len < maxKeyLength {
		k.len = maxKeyLength
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
	return dataType(k.data[typePosition])
}

func (k key) GetVersion() timestamp.Nano {
	return timestamp.Nano(binary.BigEndian.Uint64(k.data[typePosition+1:]) ^ signBit)
}
