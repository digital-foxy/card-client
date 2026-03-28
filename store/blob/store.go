package blob

import (
	"context"
	"image"

	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-parser/character"
	"github.com/digital-foxy/card-parser/png"
	"github.com/digital-foxy/toolkit/timestamp"
)

// Builder creates blob stores from a path
type Builder interface {
	Build(path string) (Store, error)
}

// Store is the main blob storage interface
type Store interface {
	CtxStore

	WithContext(ctx context.Context) CtxStore
	Close() error
}

// CtxStore is a context-aware blob store
type CtxStore interface {
	TxStore

	WithReadTx(fn func(TxReadStore) error) error
	WithWriteTx(fn func(TxWriteStore) error) error
	WithTx(fn func(TxStore) error) error
}

// TxStore combines read and write operations with Put
type TxStore interface {
	TxReadStore
	TxWriteStore

	Put(rid resource.RID, version timestamp.Nano, characterCard *png.CharacterCard) error
}

// TxReadStore provides read operations for blobs
type TxReadStore interface {
	RIDs() ([]resource.RID, error)
	GetCharacterCard(rid resource.RID, version timestamp.Nano) (*png.CharacterCard, error)
	GetRawCard(rid resource.RID, version timestamp.Nano) (*png.RawCard, error)
	GetRawCardBytes(rid resource.RID, version timestamp.Nano) ([]byte, error)
	GetSheet(rid resource.RID, version timestamp.Nano) (*character.Sheet, error)
	GetSheetBytes(rid resource.RID, version timestamp.Nano) ([]byte, error)
	Thumbnail(rid resource.RID) (image.Image, error)
	ThumbnailBytes(rid resource.RID) ([]byte, error)
	Versions(rid resource.RID) []timestamp.Nano
	VersionExists(rid resource.RID, version timestamp.Nano) (bool, error)
}

// TxWriteStore provides write/delete operations for blobs
type TxWriteStore interface {
	DeleteVersion(rid resource.RID, version timestamp.Nano) error
	DeleteVersions(rid resource.RID, lower timestamp.Nano, upper timestamp.Nano) error
	Delete(rid resource.RID) error
}
