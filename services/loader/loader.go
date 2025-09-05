package loader

import (
	"github.com/r3dpixel/card-client/services/store"
)

type Service interface {
	LoadVault(vault string) (store.Service, error)
}
