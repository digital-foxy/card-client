package client

import (
	"github.com/r3dpixel/card-client/credentials"
	"github.com/r3dpixel/card-client/credentials/keyringcred"
	"github.com/r3dpixel/card-client/facade"
	"github.com/r3dpixel/card-client/library"
	"github.com/r3dpixel/card-client/library/fslibrary"
	"github.com/r3dpixel/card-client/operation"
	"github.com/r3dpixel/card-client/operation/opcache"
	"github.com/r3dpixel/card-client/preferences"
	"github.com/r3dpixel/card-client/preferences/viper"
	"github.com/r3dpixel/card-client/tracker/mutracker"
	"github.com/r3dpixel/card-fetcher/fetcher"
	"github.com/r3dpixel/card-fetcher/impl"
	"github.com/r3dpixel/card-fetcher/router"
	"github.com/r3dpixel/toolkit/trace"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ServiceManager struct {
	PreferencesService preferences.Service
	CredentialsService credentials.Service
	OperationsService  operation.Service
	LibraryService     library.Handler
	FacadeService      facade.Service
}

func NewManager(config Config) (*ServiceManager, error) {
	sm := &ServiceManager{}

	credentialsService := keyringcred.NewService()
	preferencesService := viper.NewService(config.Preferences)
	trackerService := mutracker.New()
	registryService := opcache.NewRegistry(opcache.DefaultIdGenerator())

	libraryService, err := fslibrary.NewFsLibrary(config.Library)
	if err != nil {
		return nil, err
	}

	builders := []fetcher.Builder{
		impl.CharacterTavernBuilder{},
		impl.ChubAIBuilder{},
		impl.NyaiMeBuilder{},
		impl.PephopBuilder{},
		impl.PygmalionBuilder{IdentityReader: sm.CredentialsService.GetReader(credentials.Pygmalion)},
		impl.WyvernChatBuilder{},
	}
	routerService := router.New(config.Http)
	routerService.RegisterBuilders(builders...)

	sm.CredentialsService = credentialsService
	sm.PreferencesService = preferencesService
	sm.OperationsService = registryService
	sm.LibraryService = libraryService
	sm.FacadeService = facade.NewService(preferencesService, trackerService, registryService, libraryService, routerService)

	return sm, nil
}

func (sm *ServiceManager) InitConsoleLogging() {
	zerolog.ErrorMarshalFunc = trace.ErrorMarshalFunc
	log.Logger = log.Logger.Output(trace.ConsoleTraceWriter())
}
