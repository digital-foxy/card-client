package blob

import (
	"image"

	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/timestamp"
)

type Store interface {
	Get(rid resource.RID, version timestamp.Nano) (*png.RawCard, error)
	LoadThumbnail(rid resource.RID) (image.Image, error)
	Versions(rid resource.RID) []timestamp.Nano
	Put(rid resource.RID, version timestamp.Nano, rawCard *png.RawCard) error
	DeleteVersion(rid resource.RID, version timestamp.Nano) error
	Delete(rid resource.RID) error
}
