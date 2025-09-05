package vault

import (
	"os"
	"path/filepath"

	"github.com/r3dpixel/card-client/services/vault"
	"github.com/r3dpixel/toolkit/trace"
	"github.com/rs/zerolog/log"
)

type repository struct {
	rootDir string
}

func newRepository(rootDir string) *repository {
	return &repository{
		rootDir: rootDir,
	}
}

func (r *repository) All() map[string]vault.Vault {
	vaults := make(map[string]vault.Vault)
	dirs, err := os.ReadDir(r.rootDir)
	if err != nil {
		log.Error().Err(err).
			Str(trace.SERVICE, "vault").
			Str(trace.PATH, r.rootDir).
			Msg("Failed to read directory")
		return vaults
	}

	for _, dir := range dirs {
		v := newVault(r.rootDir, dir.Name())
		if !isValid(&v) {
			_ = r.DeleteVault(v.Name)
		} else {
			vaults[dir.Name()] = v
		}
	}

	return vaults
}

func (r *repository) CreateVault(name string) (vault.Vault, error) {
	v := newVault(r.rootDir, name)
	if isValid(&v) {
		return v, vault.ErrVaultAlreadyExists
	}

	if err := r.DeleteVault(name); err != nil {
		return vault.Vault{}, trace.Err().Wrap(err).Field("vault", name).Msg("Failed to cleanup for creation")
	}

	if err := createPaths(&v); err != nil {
		_ = r.DeleteVault(name)
		return vault.Vault{}, trace.Err().Wrap(err).Field("vault", name).Msg("Failed to create paths")
	}

	return v, nil
}

func (r *repository) DeleteVault(name string) error {
	return os.RemoveAll(filepath.Join(r.rootDir, name))
}
