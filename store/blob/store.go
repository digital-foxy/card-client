package blob

import (
	"context"
	"image"

	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/timestamp"
)

type Store interface {
	ReadStore
	WriteStore

	WithContext(ctx context.Context) Store
	WithReadTx(fn func(store ReadStore) error) error
	WithWriteTx(fn func(store WriteStore) error) error
	WithReadWriteTx(fn func(Store) error) error
	Close() error
}

type ReadStore interface {
	Get(rid resource.RID, version timestamp.Nano) (*png.RawCard, error)
	Thumbnail(rid resource.RID) (image.Image, error)
	Versions(rid resource.RID) []timestamp.Nano
}

type WriteStore interface {
	Put(rid resource.RID, version timestamp.Nano, rawCard *png.RawCard) error
	DeleteVersion(rid resource.RID, version timestamp.Nano) error
	Delete(rid resource.RID) error
}
