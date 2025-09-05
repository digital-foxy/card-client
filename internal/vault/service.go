package vault

import (
	"maps"
	"slices"
	"sync"

	"github.com/r3dpixel/card-client/opts"
	"github.com/r3dpixel/card-client/services/vault"
)

const defaultVaultLimit = 50

type Service struct {
	mutex      sync.RWMutex
	repository *repository
	vaultLimit int
	vaults     map[string]vault.Vault
}

func NewService(opts opts.VaultOptions) *Service {
	r := newRepository(opts.RootDir)
	if opts.VaultLimit <= 0 {
		opts.VaultLimit = defaultVaultLimit
	}
	s := &Service{
		repository: r,
		vaultLimit: opts.VaultLimit,
		vaults:     r.All(),
	}
	return s
}

func (s *Service) VaultCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.vaults)
}

func (s *Service) GetVaults() []vault.Vault {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return slices.Collect(maps.Values(s.vaults))
}

func (s *Service) GetVaultNames() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return slices.Collect(maps.Keys(s.vaults))
}

func (s *Service) GetVault(vaultName string) (vault.Vault, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	v, exists := s.vaults[vaultName]
	if !exists {
		return vault.Vault{}, false
	}
	return v, true
}

func (s *Service) CreateVault(name string) (vault.Vault, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if v, exists := s.vaults[name]; exists {
		return v, vault.ErrVaultAlreadyExists
	}

	if len(s.vaults) >= s.vaultLimit {
		return vault.Vault{}, vault.ErrVaultLimitExceeded
	}

	v, err := s.repository.CreateVault(name)
	if err != nil {
		return vault.Vault{}, err
	}

	s.vaults[name] = v
	return v, nil
}

func (s *Service) DeleteVault(vaultName string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.vaults[vaultName]; !exists {
		return nil
	}

	if err := s.repository.DeleteVault(vaultName); err != nil {
		return err
	}

	delete(s.vaults, vaultName)
	return nil
}
