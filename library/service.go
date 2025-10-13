package library

import (
	"github.com/r3dpixel/card-client/store/blob"
	"github.com/r3dpixel/card-client/store/catalog"
	"github.com/r3dpixel/card-client/store/record"
)

const (
	RecordPath   = "records"
	BlobPath     = "cards"
	ManifestPath = "manifest"
)

type VaultName string
type VaultPath string

type RecordStorage string

const (
	EntSQL RecordStorage = "erecord"
)

type BlobStorage string

const (
	Pebble BlobStorage = "pblob"
)

type Manifest struct {
	RecordStorage RecordStorage
	BlobStorage   BlobStorage
}

type Vault struct {
	Name    VaultName
	Catalog catalog.Service
}

type Options struct {
	Path           string
	MaxVaults      int
	RecordStorage  RecordStorage
	BlobStorage    BlobStorage
	RecordStorages map[RecordStorage]record.Builder
	BlobStorages   map[BlobStorage]blob.Builder
}

type Service interface {
	Handler
	Load(name VaultName) (Vault, error)
}

type Handler interface {
	Names() []VaultName
	Count() int
	Create(name VaultName) error
	Delete(name VaultName) error
}
