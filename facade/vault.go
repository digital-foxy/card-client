package facade

import (
	"sync"

	"github.com/digital-foxy/card-client/library"
	"github.com/digital-foxy/toolkit/trace"
	"github.com/rs/zerolog/log"
)

// vaultManager handles vault loading and locking
type vaultManager struct {
	library library.Service
	vault   library.Vault
	vaultMu sync.RWMutex
}

func newVaultManager(library library.Service) *vaultManager {
	return &vaultManager{
		library: library,
	}
}

func (m *vaultManager) LoadVault(name library.VaultName) error {
	if !m.vaultMu.TryLock() {
		return trace.Error().Msg("Vault is busy")
	}
	defer m.vaultMu.Unlock()

	vault, err := m.library.Load(name)
	if err != nil {
		log.Error().Err(err).
			Str(trace.SERVICE, "vault").
			Str(trace.ACTIVITY, "load-vault").
			Str("vault", string(name)).
			Msg("Could not load vault")
		return trace.Error().Msg("Could not load vault")
	}

	oldCatalog := m.vault.Catalog
	m.vault = vault

	if oldCatalog != nil {
		err = oldCatalog.Close()
		log.Warn().Err(err).
			Str(trace.SERVICE, "vault").
			Str(trace.ACTIVITY, "load-vault").
			Str("vault", string(m.vault.Name)).
			Msg("Could not close old store service")
	}

	return nil
}

func (m *vaultManager) UnloadVault() error {
	if !m.vaultMu.TryLock() {
		return trace.Error().Msg("Vault is busy")
	}
	defer m.vaultMu.Unlock()

	if m.vault.Catalog != nil {
		if err := m.vault.Catalog.Close(); err != nil {
			log.Warn().Err(err).
				Str(trace.SERVICE, "vault").
				Str(trace.ACTIVITY, "unload-vault").
				Msg("Could not unload vault")
		}
	}

	m.vault.Catalog = nil
	m.vault.Name = ""
	return nil
}

func (m *vaultManager) beginReadOp() (vault library.Vault, unlock func(), err error) {
	if !m.vaultMu.TryRLock() {
		return library.Vault{}, nil, trace.Error().Msg("Vault is loading")
	}

	if m.vault.Catalog == nil {
		m.vaultMu.RUnlock()
		return library.Vault{}, nil, trace.Error().Msg("No vault loaded")
	}

	return m.vault, m.vaultMu.RUnlock, nil
}
