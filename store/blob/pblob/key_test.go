package pblob

import (
	"testing"

	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/toolkit/timestamp"
	"github.com/stretchr/testify/assert"
)

func TestKeyBuilding(t *testing.T) {
	rid := resource.RID(12345)
	version := timestamp.Nano(67890)

	tests := []struct {
		name        string
		keyFunc     func() []byte
		expectedLen int
		expectedTyp dataType
		hasVersion  bool
	}{
		{
			name:        "pngKey",
			keyFunc:     func() []byte { return newKey().RID(rid).Type(pngType).Version(version).Bytes() },
			expectedLen: 17,
			expectedTyp: pngType,
			hasVersion:  true,
		},
		{
			name:        "jsonKey",
			keyFunc:     func() []byte { return newKey().RID(rid).Type(jsonType).Version(version).Bytes() },
			expectedLen: 17,
			expectedTyp: jsonType,
			hasVersion:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.keyFunc()
			assert.Len(t, key, tt.expectedLen)

			k := keyOf(key)
			assert.Equal(t, rid, k.GetRID())
			assert.Equal(t, tt.expectedTyp, k.GetType())
			if tt.hasVersion {
				assert.Equal(t, version, k.GetVersion())
			}
		})
	}
}

func TestPrefixes(t *testing.T) {
	rid := resource.RID(12345)

	tests := []struct {
		name        string
		prefixFunc  func() []byte
		expectedLen int
		expectedTyp dataType
	}{
		{
			name:        "pngPrefix",
			prefixFunc:  func() []byte { return newKey().RID(rid).Type(pngType).Bytes() },
			expectedLen: 9,
			expectedTyp: pngType,
		},
		{
			name:        "jsonPrefix",
			prefixFunc:  func() []byte { return newKey().RID(rid).Type(jsonType).Bytes() },
			expectedLen: 9,
			expectedTyp: jsonType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix := tt.prefixFunc()
			assert.Len(t, prefix, tt.expectedLen)

			k := keyOf(prefix)
			assert.Equal(t, rid, k.GetRID())
			assert.Equal(t, tt.expectedTyp, k.GetType())
		})
	}
}

func TestThumbnailKey(t *testing.T) {
	rid := resource.RID(12345)

	tk := newKey().RID(rid).Type(thumbnailType).Bytes()
	assert.Len(t, tk, 9)

	k := keyOf(tk)
	assert.Equal(t, rid, k.GetRID())
	assert.Equal(t, thumbnailType, k.GetType())
}

func TestKeyOrderIndependent(t *testing.T) {
	rid := resource.RID(12345)
	version := timestamp.Nano(67890)

	k1 := newKey().RID(rid).Type(pngType).Version(version)
	k2 := newKey().Version(version).Type(pngType).RID(rid)
	k3 := newKey().Type(pngType).RID(rid).Version(version)

	assert.Equal(t, k1, k2)
	assert.Equal(t, k1, k3)
}

func TestVersionSortOrder(t *testing.T) {
	rid := resource.RID(1)
	v1 := timestamp.Nano(-100)
	v2 := timestamp.Nano(-50)
	v3 := timestamp.Nano(0)
	v4 := timestamp.Nano(50)
	v5 := timestamp.Nano(100)

	k1 := newKey().RID(rid).Type(pngType).Version(v1).Bytes()
	k2 := newKey().RID(rid).Type(pngType).Version(v2).Bytes()
	k3 := newKey().RID(rid).Type(pngType).Version(v3).Bytes()
	k4 := newKey().RID(rid).Type(pngType).Version(v4).Bytes()
	k5 := newKey().RID(rid).Type(pngType).Version(v5).Bytes()

	keys := [][]byte{k1, k2, k3, k4, k5}
	for i := 0; i < len(keys)-1; i++ {
		assert.True(t, lessThan(keys[i], keys[i+1]), "key[%d] should be less than key[%d]", i, i+1)
	}
}

func lessThan(a, b []byte) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] < b[i] {
			return true
		}
		if a[i] > b[i] {
			return false
		}
	}
	return len(a) < len(b)
}

func TestRoundTrip(t *testing.T) {
	testCases := []struct {
		name    string
		rid     resource.RID
		version timestamp.Nano
	}{
		{"zero values", 0, 0},
		{"small values", 1, 1},
		{"medium values", 123456789, 987654321},
		{"max values", ^resource.RID(0), ^timestamp.Nano(0)},
	}

	keyFuncs := []struct {
		name string
		fn   func(resource.RID, timestamp.Nano) []byte
		typ  dataType
	}{
		{"pngKey", func(rid resource.RID, v timestamp.Nano) []byte {
			return newKey().RID(rid).Type(pngType).Version(v).Bytes()
		}, pngType},
		{"jsonKey", func(rid resource.RID, v timestamp.Nano) []byte {
			return newKey().RID(rid).Type(jsonType).Version(v).Bytes()
		}, jsonType},
	}

	for _, kf := range keyFuncs {
		t.Run(kf.name, func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					key := kf.fn(tc.rid, tc.version)
					k := keyOf(key)

					assert.Equal(t, tc.rid, k.GetRID())
					assert.Equal(t, kf.typ, k.GetType())
					assert.Equal(t, tc.version, k.GetVersion())
				})
			}
		})
	}
}

func TestTypeSortOrder(t *testing.T) {
	rid := resource.RID(1)
	version := timestamp.Nano(100)

	// Keys should be ordered: png (0x00) < json (0x02) < thumbnail (0x80)
	pngK := newKey().RID(rid).Type(pngType).Version(version).Bytes()
	jsonK := newKey().RID(rid).Type(jsonType).Version(version).Bytes()
	thumbK := newKey().RID(rid).Type(thumbnailType).Bytes()

	assert.True(t, lessThan(pngK[:9], jsonK[:9]), "png type should be less than json type")
	assert.True(t, lessThan(jsonK[:9], thumbK), "json type should be less than thumbnail type")
	assert.True(t, lessThan(pngK[:9], thumbK), "png type should be less than thumbnail type")
}

func TestKeyBytes(t *testing.T) {
	rid := resource.RID(12345)
	version := timestamp.Nano(67890)

	tests := []struct {
		name        string
		key         key
		expectedLen int
	}{
		{
			name:        "RID only",
			key:         newKey().RID(rid),
			expectedLen: 8,
		},
		{
			name:        "RID and type",
			key:         newKey().RID(rid).Type(pngType),
			expectedLen: 9,
		},
		{
			name:        "full key",
			key:         newKey().RID(rid).Type(pngType).Version(version),
			expectedLen: 17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes := tt.key.Bytes()
			assert.Len(t, bytes, tt.expectedLen)
		})
	}
}
