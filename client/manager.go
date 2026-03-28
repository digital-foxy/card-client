package client

import (
	"github.com/digital-foxy/card-client/credentials"
	"github.com/digital-foxy/card-client/credentials/keyringcred"
	"github.com/digital-foxy/card-client/facade"
	"github.com/digital-foxy/card-client/library/fslibrary"
	"github.com/digital-foxy/card-client/operation"
	"github.com/digital-foxy/card-client/operation/opcache"
	"github.com/digital-foxy/card-client/preferences"
	"github.com/digital-foxy/card-client/preferences/vpref"
	"github.com/digital-foxy/card-client/tracker/mutracker"
	"github.com/digital-foxy/card-fetcher/impl"
	"github.com/digital-foxy/card-fetcher/router"
	"github.com/digital-foxy/toolkit/trace"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ServiceManager provides access to all application services
type ServiceManager struct {
	PreferencesService preferences.Service
	CredentialsService credentials.Service
	OperationsService  operation.Service
	LibraryService     *fslibrary.FsLibrary
	FacadeService      facade.Service
}

// NewManager initializes all services with the given config
func NewManager(config Config) (*ServiceManager, error) {
	if config.Workers <= 0 {
		config.Workers = 1
	}

	sm := &ServiceManager{}

	credentialsService := keyringcred.NewService()
	preferencesService := vpref.NewService(config.Preferences)
	trackerService := mutracker.New()
	registryService := opcache.NewRegistry(opcache.DefaultIdGenerator())

	libraryService, err := fslibrary.NewFsLibrary(config.Library, registryService)
	if err != nil {
		return nil, err
	}

	builders := impl.DefaultBuilders(
		impl.BuilderOptions{
			PygmalionIdentityReader: credentialsService.GetReader(credentials.Pygmalion),
			ChromePath: func() string {
				return preferencesService.GetString(preferences.ChromePath)
			},
		},
	)
	routerService := router.New(config.Http)
	routerService.RegisterBuilders(builders...)

	sm.CredentialsService = credentialsService
	sm.PreferencesService = preferencesService
	sm.OperationsService = registryService
	sm.LibraryService = libraryService
	sm.FacadeService = facade.NewService(
		preferencesService,
		trackerService,
		registryService,
		libraryService,
		routerService,
		config.UploadThumbnailSize,
		config.Workers,
	)

	return sm, nil
}

func (sm *ServiceManager) InitConsoleLogging() {
	zerolog.ErrorMarshalFunc = trace.ErrorMarshalFunc
	log.Logger = log.Logger.Output(trace.ConsoleTraceWriter())
}
