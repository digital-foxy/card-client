package vault

import (
	"errors"

	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/toolkit/timestamp"
)

var ErrVaultAlreadyExists = errors.New("Vault already exists")
var ErrVaultLimitExceeded = errors.New("Vault limit exceeded")

type Vault struct {
	Name         string
	VaultDir     string
	CardsDir     string
	DbFilePath   string
	ConfFilePath string
}

type Service interface {
	VaultCount() int
	GetVaults() []Vault
	GetVaultNames() []string
	GetVault(name string) (Vault, bool)
	CreateVault(name string) (Vault, error)
	DeleteVault(name string) error
}

type Stats struct {
	NoCards    int64
	NoCreators int64
	NoTags     int64
	Sources    map[source.ID]int64
	Created    timestamp.Milli
}
