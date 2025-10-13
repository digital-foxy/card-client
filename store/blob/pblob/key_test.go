package pblob

import (
	"testing"

	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/toolkit/timestamp"
	"github.com/stretchr/testify/assert"
)

func TestKeyBuilding(t *testing.T) {
	rid := resource.RID(12345)
	version := timestamp.Nano(67890)

	vk := versionKey(rid, version)
	assert.Len(t, vk, 17)

	k := keyOf(vk)
	assert.Equal(t, rid, k.GetRID())
	assert.Equal(t, versionType, k.GetType())
	assert.Equal(t, version, k.GetVersion())
}

func TestVersionPrefix(t *testing.T) {
	rid := resource.RID(12345)

	vp := versionPrefix(rid)
	assert.Len(t, vp, 9)

	k := keyOf(vp)
	assert.Equal(t, rid, k.GetRID())
	assert.Equal(t, versionType, k.GetType())
}

func TestThumbnailKey(t *testing.T) {
	rid := resource.RID(12345)

	tk := thumbnailKey(rid)
	assert.Len(t, tk, 9)

	k := keyOf(tk)
	assert.Equal(t, rid, k.GetRID())
	assert.Equal(t, thumbnailType, k.GetType())
}

func TestKeyOrderIndependent(t *testing.T) {
	rid := resource.RID(12345)
	version := timestamp.Nano(67890)

	k1 := newKey().RID(rid).Type(versionType).Version(version)
	k2 := newKey().Version(version).Type(versionType).RID(rid)
	k3 := newKey().Type(versionType).RID(rid).Version(version)

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

	k1 := versionKey(rid, v1)
	k2 := versionKey(rid, v2)
	k3 := versionKey(rid, v3)
	k4 := versionKey(rid, v4)
	k5 := versionKey(rid, v5)

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
		rid     resource.RID
		version timestamp.Nano
	}{
		{0, 0},
		{1, 1},
		{123456789, 987654321},
		{^resource.RID(0), ^timestamp.Nano(0)},
	}

	for _, tc := range testCases {
		vk := versionKey(tc.rid, tc.version)
		k := keyOf(vk)

		assert.Equal(t, tc.rid, k.GetRID())
		assert.Equal(t, tc.version, k.GetVersion())
	}
}
