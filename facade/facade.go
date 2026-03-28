package facade

import (
	"github.com/digital-foxy/card-client/library"
	"github.com/digital-foxy/card-client/operation"
	"github.com/digital-foxy/card-client/preferences"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-client/tracker"
	"github.com/digital-foxy/card-fetcher/router"
	"github.com/digital-foxy/card-fetcher/source"
	"github.com/digital-foxy/card-parser/character"
	"github.com/digital-foxy/toolkit/timestamp"
)

// OperationPayload contains operation ID and associated record
type OperationPayload struct {
	OperationID operation.ID
	Record      resource.Record
}

// IntegrationReport contains source integration status
type IntegrationReport struct {
	SourceID source.ID
	Status   router.IntegrationStatus
}

// Service is the main API for all card operations
type Service interface {
	LoadVault(vault library.VaultName) error
	UnloadVault() error

	CountRecords(filter resource.Filter) (int, error)
	FindRIDs(filter resource.Filter) ([]resource.RID, error)
	FindPagedIDs(filter resource.Filter, offset int, limit int) ([]resource.RID, error)
	FindRecords(rids ...resource.RID) (resource.Box[resource.Record], error)
	Sheet(rid resource.RID, version timestamp.Nano) (*character.Sheet, error)
	ThumbnailBytes(rid resource.RID) ([]byte, error)
	TagNames() ([]string, error)

	ImportURLs(rawURLs string) (operation.ID, error)
	ExportLatestCards(rids ...resource.RID) (operation.ID, error)
	ExportVault() (operation.ID, error)
	UpdateCards(force bool, rids ...resource.RID) (operation.ID, error)
	CheckIntegrity() (operation.ID, error)

	ToggleFavorite(rid resource.RID) error
	UpdateFavorites(favorite bool, rids ...resource.RID) error

	Flush() (resource.Box[OperationPayload], error)
	HasRequests() bool

	LoadLocalCard(path string) (UploadRequest, error)
	UnloadLocalCard(signature Signature)
	AcceptLocalCard(response UploadResponse) error

	GetSources() []source.ID
	GetSourceStatus(sourceID source.ID) router.IntegrationStatus
	GetFilterControls() resource.FieldControls
}

// Facade implements Service by composing specialized services
type Facade struct {
	*vaultManager
	*queryService
	*syncService
	*exportService
	*favoriteService
	*cacheManager
	*uploadService
	router *router.Router
}

// NewService creates a new Facade with all dependencies
func NewService(
	pref preferences.Service,
	tracker tracker.Service,
	registry operation.Registry,
	library library.Service,
	router *router.Router,
	uploadThumbnailSize int,
	workers int,
) *Facade {
	vaultMgr := newVaultManager(library)
	cacheMgr := newCacheManager(vaultMgr)

	return &Facade{
		vaultManager:    vaultMgr,
		queryService:    newQueryService(vaultMgr, router),
		syncService:     newSyncService(vaultMgr, tracker, registry, router, cacheMgr, workers),
		exportService:   newExportService(vaultMgr, tracker, pref, registry, cacheMgr, workers),
		favoriteService: newFavoriteService(vaultMgr),
		cacheManager:    cacheMgr,
		uploadService:   newUploadService(uploadThumbnailSize),
		router:          router,
	}
}

func (f *Facade) GetSources() []source.ID {
	return f.router.Sources()
}

func (f *Facade) GetSourceStatus(sourceID source.ID) router.IntegrationStatus {
	return f.router.CheckIntegration(sourceID)
}
