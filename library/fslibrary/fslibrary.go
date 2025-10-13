package fslibrary

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/r3dpixel/card-client/library"
	"github.com/r3dpixel/card-client/store/blob"
	"github.com/r3dpixel/card-client/store/catalog"
	"github.com/r3dpixel/card-client/store/record"
	"github.com/r3dpixel/toolkit/filex"
	"github.com/r3dpixel/toolkit/jsonx"
)

type FsLibrary struct {
	root           string
	maxVaults      int
	recordStorage  library.RecordStorage
	blobStorage    library.BlobStorage
	recordStorages map[library.RecordStorage]record.Builder
	blobStorages   map[library.BlobStorage]blob.Builder
	mu             sync.RWMutex
	vaults         map[library.VaultName]library.VaultPath
}

func NewFsLibrary(opts library.Options) (library.Service, error) {
	l := &FsLibrary{
		root:           opts.Path,
		maxVaults:      opts.MaxVaults,
		recordStorage:  opts.RecordStorage,
		blobStorage:    opts.BlobStorage,
		recordStorages: opts.RecordStorages,
		blobStorages:   opts.BlobStorages,
		vaults:         make(map[library.VaultName]library.VaultPath),
	}

	dirEntries, err := os.ReadDir(l.root)
	if err != nil {
		return nil, err
	}

	for _, entry := range dirEntries {
		entryPath := filepath.Join(l.root, entry.Name())
		if !entry.IsDir() {
			_ = os.RemoveAll(entryPath)
			continue
		}

		filename := entry.Name()
		l.vaults[library.VaultName(filename)] = library.VaultPath(entryPath)
	}

	return l, nil
}

func (f *FsLibrary) Names() []library.VaultName {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return slices.Collect(maps.Keys(f.vaults))
}

func (f *FsLibrary) Count() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.vaults)
}

func (f *FsLibrary) Create(name library.VaultName) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.vaults) >= f.maxVaults {
		return errors.New("maximum vaults reached")
	}
	if _, ok := f.vaults[name]; ok {
		return errors.New("vault already exists")
	}

	vaultRoot := filepath.Join(f.root, string(name))
	if err := os.MkdirAll(vaultRoot, filex.DirectoryPermission); err != nil {
		return err
	}

	f.vaults[name] = library.VaultPath(vaultRoot)
	return nil
}

func (f *FsLibrary) Load(name library.VaultName) (library.Vault, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	vaultRoot, ok := f.vaults[name]
	if !ok {
		return library.Vault{}, errors.New("vault not found")
	}
	recordPath := filepath.Join(string(vaultRoot), library.RecordPath)
	blobPath := filepath.Join(string(vaultRoot), library.BlobPath)
	manifestPath := filepath.Join(string(vaultRoot), library.ManifestPath)

	manifest := library.Manifest{
		RecordStorage: f.recordStorage,
		BlobStorage:   f.blobStorage,
	}
	err := jsonx.ToFile(manifest, manifestPath)
	if err != nil {
		return library.Vault{}, err
	}

	recordStorage, ok := f.recordStorages[f.recordStorage]
	if !ok {
		return library.Vault{}, errors.New("record storage type not found")
	}

	blobStorage, ok := f.blobStorages[f.blobStorage]
	if !ok {
		return library.Vault{}, errors.New("blob storage type not found")
	}

	recordStore, err := recordStorage.Build(recordPath)
	if err != nil {
		return library.Vault{}, err
	}

	blobStore, err := blobStorage.Build(blobPath)
	if err != nil {
		return library.Vault{}, err
	}

	vault := library.Vault{
		Name:    name,
		Catalog: catalog.New(recordStore, blobStore),
	}

	return vault, nil
}

func (f *FsLibrary) Delete(name library.VaultName) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := os.RemoveAll(filepath.Join(f.root, string(name))); err != nil {
		return err
	}
	delete(f.vaults, name)

	return nil
}
