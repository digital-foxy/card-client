package vault

import (
	"os"
	"path/filepath"

	"github.com/r3dpixel/card-client/services/vault"
	"github.com/r3dpixel/toolkit/filex"
)

const (
	cardsDirName string = "cards"
	dbFileName   string = "entries"
	confFileName string = "conf"
)

func newVault(rootDir string, name string) vault.Vault {
	return vault.Vault{
		Name:         name,
		VaultDir:     filepath.Join(rootDir, name),
		CardsDir:     filepath.Join(rootDir, name, cardsDirName),
		DbFilePath:   filepath.Join(rootDir, name, dbFileName),
		ConfFilePath: filepath.Join(rootDir, name, confFileName),
	}
}

func isValid(v *vault.Vault) bool {
	return filex.DirExists(v.CardsDir) && filex.FileExists(v.ConfFilePath)
}

func createPaths(v *vault.Vault) error {
	if err := os.MkdirAll(v.CardsDir, filex.DirectoryPermission); err != nil {
		return err
	}
	if _, err := os.Create(v.ConfFilePath); err != nil {
		return err
	}

	return nil
}
