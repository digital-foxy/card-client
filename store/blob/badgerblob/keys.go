package badgerblob

import (
	"encoding/binary"

	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/toolkit/timestamp"
)

const maxKeyLength = 16

const (
	signBit   = 0x80
	signBit8  = signBit
	signBit16 = signBit << 8
	signBit32 = signBit << 24
	signBit64 = signBit << 56
)

func versionedKey(rid resource.RID, version timestamp.Nano) []byte {
	return (&keyBuilder{}).
		WriteUint64(uint64(rid), binary.BigEndian).
		WriteInt64(int64(version), binary.BigEndian).
		Bytes()
}

type shardedKey [maxKeyLength]byte

type keyBuilder struct {
	key shardedKey
	off int
}

func (kb *keyBuilder) WriteByte(b byte) *keyBuilder {
	kb.key[kb.off] = b
	kb.off++
	return kb
}

func (kb *keyBuilder) WriteUint8(v uint8) *keyBuilder {
	kb.key[kb.off] = v
	kb.off++
	return kb
}

func (kb *keyBuilder) WriteInt8(v int8) *keyBuilder {
	kb.key[kb.off] = uint8(v) ^ signBit8
	kb.off++
	return kb
}

func (kb *keyBuilder) WriteUint16(v uint16, order binary.ByteOrder) *keyBuilder {
	order.PutUint16(kb.key[kb.off:], v)
	kb.off += 2
	return kb
}

func (kb *keyBuilder) WriteInt16(v int16, order binary.ByteOrder) *keyBuilder {
	return kb.WriteUint16(uint16(v)^signBit16, order)
}

func (kb *keyBuilder) WriteUint32(v uint32, order binary.ByteOrder) *keyBuilder {
	order.PutUint32(kb.key[kb.off:], v)
	kb.off += 4
	return kb
}

func (kb *keyBuilder) WriteInt32(v int32, order binary.ByteOrder) *keyBuilder {
	return kb.WriteUint32(uint32(v)^signBit32, order)
}

func (kb *keyBuilder) WriteUint64(v uint64, order binary.ByteOrder) *keyBuilder {
	order.PutUint64(kb.key[kb.off:], v)
	kb.off += 8
	return kb
}

func (kb *keyBuilder) WriteInt64(v int64, order binary.ByteOrder) *keyBuilder {
	return kb.WriteUint64(uint64(v)^signBit64, order)
}

func (kb *keyBuilder) Bytes() []byte {
	return kb.key[:kb.off]
}
