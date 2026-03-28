package facade

import (
	"github.com/digital-foxy/card-client/operation"
	"github.com/digital-foxy/card-client/tracker"
	"github.com/digital-foxy/card-fetcher/router"
)

// syncService coordinates all synchronization operations for cards.
// It handles importing new cards, updating existing ones, and integrity checks.
type syncService struct {
	vault    *vaultManager
	tracker  tracker.Service
	registry operation.Registry
	router   *router.Router
	cache    *cacheManager
	workers  int
}

func newSyncService(
	vault *vaultManager,
	tracker tracker.Service,
	registry operation.Registry,
	router *router.Router,
	cache *cacheManager,
	workers int,
) *syncService {
	return &syncService{
		vault:    vault,
		tracker:  tracker,
		registry: registry,
		router:   router,
		cache:    cache,
		workers:  workers,
	}
}
