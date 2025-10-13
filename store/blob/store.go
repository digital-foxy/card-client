package blob

import (
	"context"
	"image"

	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/timestamp"
)

type Builder interface {
	Build(path string) (Store, error)
}

type Store interface {
	CtxStore

	WithContext(ctx context.Context) CtxStore
	Close() error
}

type CtxStore interface {
	TxStore

	WithReadTx(fn func(TxReadStore) error) error
	WithWriteTx(fn func(TxWriteStore) error) error
	WithTx(fn func(TxStore) error) error
}

type TxStore interface {
	TxReadStore
	TxWriteStore

	Put(rid resource.RID, version timestamp.Nano, rawCard *png.RawCard) error
}

type TxReadStore interface {
	Get(rid resource.RID, version timestamp.Nano) (*png.RawCard, error)
	GetBytes(rid resource.RID, version timestamp.Nano) ([]byte, error)
	Thumbnail(rid resource.RID) (image.Image, error)
	ThumbnailBytes(rid resource.RID) ([]byte, error)
	Versions(rid resource.RID) []timestamp.Nano
	VersionExists(rid resource.RID, version timestamp.Nano) (bool, error)
}

type TxWriteStore interface {
	DeleteVersion(rid resource.RID, version timestamp.Nano) error
	DeleteVersions(rid resource.RID, lower timestamp.Nano, upper timestamp.Nano) error
	Delete(rid resource.RID) error
}
