package fslibrary

import (
	"context"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/digital-foxy/card-client/library"
	"github.com/digital-foxy/card-client/operation"
	"github.com/digital-foxy/card-client/store/blob"
	"github.com/digital-foxy/card-client/store/catalog"
	"github.com/digital-foxy/card-client/store/record"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-parser/png"
	"github.com/digital-foxy/toolkit/filex"
	"github.com/digital-foxy/toolkit/jsonx"
	"github.com/digital-foxy/toolkit/scheduler"
)

type vaultImportContext struct {
	cat    catalog.Service
	handle operation.Handle
}

type importItem struct {
	rid      resource.RID
	rec      resource.Record
	cardPath string
}

// FsLibrary implements library.Service using filesystem
type FsLibrary struct {
	root           string
	maxVaults      int
	workers        int
	registry       operation.Registry
	recordStorage  library.RecordStorage
	blobStorage    library.BlobStorage
	recordStorages map[library.RecordStorage]record.Builder
	blobStorages   map[library.BlobStorage]blob.Builder
	mu             sync.RWMutex
	vaults         map[library.VaultName]library.VaultPath
}

// NewFsLibrary creates a new filesystem-based library
func NewFsLibrary(opts library.Options, registry operation.Registry) (*FsLibrary, error) {
	l := &FsLibrary{
		root:           opts.Path,
		maxVaults:      opts.MaxVaults,
		workers:        opts.Workers,
		registry:       registry,
		recordStorage:  opts.RecordStorage,
		blobStorage:    opts.BlobStorage,
		recordStorages: opts.RecordStorages,
		blobStorages:   opts.BlobStorages,
		vaults:         make(map[library.VaultName]library.VaultPath),
	}

	_ = os.MkdirAll(l.root, filex.DirectoryPermission)
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

	// Read existing manifest to preserve the storage types the vault was created with.
	// Only write defaults for new vaults that have no manifest yet.
	manifest, err := jsonx.FromFile[library.Manifest](manifestPath)
	if err != nil {
		manifest = library.Manifest{
			RecordStorage: f.recordStorage,
			BlobStorage:   f.blobStorage,
		}
		if writeErr := jsonx.ToFile(manifest, manifestPath); writeErr != nil {
			return library.Vault{}, writeErr
		}
	}

	recordStorage, ok := f.recordStorages[manifest.RecordStorage]
	if !ok {
		return library.Vault{}, errors.New("record storage type not found")
	}

	blobStorage, ok := f.blobStorages[manifest.BlobStorage]
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

// ImportVault imports a vault from an exported folder.
// The folder should contain an index.json and PNG files exported by ExportVault.
// Returns an operation ID that can be used to track progress.
func (f *FsLibrary) ImportVault(exportPath string) (operation.ID, error) {
	importName := filepath.Base(exportPath)
	importPath := filex.NextAvailablePath(filepath.Join(f.root, importName))
	vaultName := library.VaultName(filepath.Base(importPath))

	handle := f.registry.NewOperation(vaultName, operation.ImportVault)

	go f.runImportVaultWorker(vaultName, handle, exportPath)

	return handle.OperationID, nil
}

func (f *FsLibrary) runImportVaultWorker(vaultName library.VaultName, handle operation.Handle, exportPath string) {
	defer f.registry.MarkTerminated(handle.OperationID)

	if err := f.Create(vaultName); err != nil {
		return
	}

	vault, err := f.Load(vaultName)
	if err != nil {
		return
	}
	defer vault.Catalog.Close()

	indexPath := filepath.Join(exportPath, "index.json")
	index, err := jsonx.FromFile[map[resource.RID]resource.Record](indexPath)
	if err != nil {
		return
	}

	// Build a map of RID -> file path once before workers start
	cardFiles, err := f.buildCardFileMap(exportPath)
	if err != nil {
		return
	}

	items := make([]importItem, 0, len(index))
	for rid, rec := range index {
		if cardPath, ok := cardFiles[rid]; ok {
			items = append(items, importItem{rid: rid, rec: rec, cardPath: cardPath})
		}
	}

	_ = f.registry.MutateReport(handle.OperationID, func(report *operation.Report) {
		report.Total = len(items)
	})

	ctx := &vaultImportContext{
		cat:    vault.Catalog.WithContext(handle.Context),
		handle: handle,
	}

	scheduler.Exec(scheduler.FromSlice(items), scheduler.Options[importItem]{
		Context:     handle.Context,
		Parallelism: f.workers,
		Handler: func(_ context.Context, item importItem) {
			f.importVaultCardWithProgress(ctx, item.cardPath, item.rec)
		},
	})
}

func (f *FsLibrary) buildCardFileMap(exportPath string) (map[resource.RID]string, error) {
	entries, err := os.ReadDir(exportPath)
	if err != nil {
		return nil, err
	}

	cardFiles := make(map[resource.RID]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".png") {
			continue
		}
		// Parse RID from filename prefix (format: "123_title.png")
		underscoreIdx := strings.Index(name, "_")
		if underscoreIdx == -1 {
			continue
		}
		rid, err := strconv.ParseInt(name[:underscoreIdx], 10, 64)
		if err != nil {
			continue
		}
		cardFiles[resource.RID(rid)] = filepath.Join(exportPath, name)
	}

	return cardFiles, nil
}

func (f *FsLibrary) importVaultCardWithProgress(ctx *vaultImportContext, cardPath string, rec resource.Record) {
	err := f.importVaultCard(ctx.cat, cardPath, rec)
	_ = f.registry.MutateReport(ctx.handle.OperationID, func(report *operation.Report) {
		report.Progress++
		if err == nil {
			report.NoSuccesses++
		} else {
			report.AuxData = append(report.AuxData, rec.Title)
			report.NoFailures++
		}
	})
}

func (f *FsLibrary) importVaultCard(cat catalog.Service, cardPath string, rec resource.Record) error {
	cardBytes, err := os.ReadFile(cardPath)
	if err != nil {
		return err
	}

	rawCard, err := png.FromBytes(cardBytes).First().Get()
	if err != nil {
		return err
	}

	characterCard, err := rawCard.Decode()
	if err != nil {
		return err
	}

	_, err = cat.RestoreCard(&rec, characterCard)
	return err
}
