package facade

import (
	"sync"

	"github.com/r3dpixel/card-client/cache"
	"github.com/r3dpixel/card-client/library"
	"github.com/r3dpixel/card-client/operation"
	"github.com/r3dpixel/card-client/preferences"
	"github.com/r3dpixel/card-client/tracker"
	"github.com/r3dpixel/card-fetcher/router"
	"github.com/r3dpixel/toolkit/stringsx"
	"github.com/r3dpixel/toolkit/trace"
	"github.com/rs/zerolog/log"
)

type Facade struct {
	builderMutex    sync.Mutex
	fileNameBuilder FileNameBuilder

	updateRequestCache *cache.RequestCache
	exportRequestCache *cache.RequestCache

	preferences preferences.Service
	tracker     tracker.Service
	registry    operation.Registry
	library     library.Service
	router      *router.Router

	vault   library.Vault
	vaultMu sync.RWMutex
}

func NewService(
	pref preferences.Service,
	tracker tracker.Service,
	registry operation.Registry,
	library library.Service,
	router *router.Router,
) *Facade {
	return &Facade{
		updateRequestCache: cache.NewRequestCache(0),
		exportRequestCache: cache.NewRequestCache(0),
		preferences:        pref,
		tracker:            tracker,
		registry:           registry,
		library:            library,
		router:             router,
	}
}

func (f *Facade) SetFileNameBuilder(builder FileNameBuilder) {
	f.builderMutex.Lock()
	defer f.builderMutex.Unlock()

	f.fileNameBuilder = builder
}

func (f *Facade) LoadVault(name library.VaultName) error {
	if !f.vaultMu.TryLock() {
		return trace.Err().Msg("Vault is busy")
	}
	defer f.vaultMu.Unlock()

	vault, err := f.library.Load(name)
	if err != nil {
		log.Error().Err(err).
			Str(trace.SERVICE, "facade").
			Str(trace.ACTIVITY, "load-vault").
			Str("vault", string(name)).
			Msg("Could not load vault")
		return trace.Err().Msg("Could not load vault")
	}

	oldCatalog := f.vault.Catalog
	f.vault = vault

	if oldCatalog != nil {
		err = oldCatalog.Close()
		log.Warn().Err(err).
			Str(trace.SERVICE, "facade").
			Str(trace.ACTIVITY, "load-vault").
			Str("vault", string(f.vault.Name)).
			Msg("Could not close old store service")
	}

	return nil
}

func (f *Facade) UnloadVault() error {
	if !f.vaultMu.TryLock() {
		return trace.Err().Msg("Vault is busy")
	}
	defer f.vaultMu.Unlock()

	if f.vault.Catalog != nil {
		err := f.vault.Catalog.Close()
		log.Warn().Err(err).
			Str(trace.SERVICE, "facade").
			Str(trace.ACTIVITY, "unload-vault").
			Msg("Could not unload vault")
	}

	f.vault.Catalog = nil
	f.vault.Name = library.VaultName(stringsx.Empty)
	return nil
}

func (f *Facade) beginReadStoreOp() (unlock func(), err error) {
	if !f.vaultMu.TryRLock() {
		return nil, trace.Err().Msg("Vault is loading")
	}

	if f.vault.Catalog == nil {
		f.vaultMu.RUnlock()
		return nil, trace.Err().Msg("No vault loaded")
	}

	return f.vaultMu.RUnlock, nil
}
