package library

import (
	"github.com/digital-foxy/card-client/store/blob"
	"github.com/digital-foxy/card-client/store/catalog"
	"github.com/digital-foxy/card-client/store/record"
)

const (
	RecordPath   = "records"
	BlobPath     = "cards"
	ManifestPath = "manifest"
)

// VaultName is a unique identifier for a vault
type VaultName string

// VaultPath is the filesystem path to a vault
type VaultPath string

// RecordStorage identifies a record storage implementation
type RecordStorage string

const (
	EntSQL RecordStorage = "erecord"
)

// BlobStorage identifies a blob storage implementation
type BlobStorage string

const (
	Pebble     BlobStorage = "pblob"
	SQLiteBlob BlobStorage = "sqlblob"
)

// Manifest describes the storage types used by a vault
type Manifest struct {
	RecordStorage RecordStorage
	BlobStorage   BlobStorage
}

// Vault represents a loaded vault with its catalog
type Vault struct {
	Name    VaultName
	Catalog catalog.Service
}

// Options configures the library service
type Options struct {
	Path           string
	MaxVaults      int
	Workers        int
	RecordStorage  RecordStorage
	BlobStorage    BlobStorage
	RecordStorages map[RecordStorage]record.Builder
	BlobStorages   map[BlobStorage]blob.Builder
}

// Service manages multiple vaults
type Service interface {
	Handler
	Load(name VaultName) (Vault, error)
}

// Handler provides vault management operations
type Handler interface {
	Names() []VaultName
	Count() int
	Create(name VaultName) error
	Delete(name VaultName) error
}
