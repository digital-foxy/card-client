package manager

import (
	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/r3dpixel/card-client/credentials"
	"github.com/r3dpixel/card-client/credentials/keyringcred"
	"github.com/r3dpixel/card-client/facade/facade"
	"github.com/r3dpixel/card-client/operation"
	"github.com/r3dpixel/card-client/operation/opcache"
	"github.com/r3dpixel/card-client/opts"
	"github.com/r3dpixel/card-client/preferences"
	"github.com/r3dpixel/card-client/preferences/viperpref"
	"github.com/r3dpixel/card-client/store/record/entrecord/ent"
	"github.com/r3dpixel/card-client/tracker/mutracker"
	"github.com/r3dpixel/card-fetcher/router"
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/toolkit/trace"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ServiceManager struct {
	PreferencesService preferences.Service
	CredentialsService credentials.Service
	OperationsService  operation.Service
	FacadeService      facade.Service
}

func NewManager(config opts.AppConfig) *ServiceManager {
	sm := &ServiceManager{}

	credentialsService := keyringcred.NewService()
	preferencesService := viperpref.NewService()
	trackerService := mutracker.NewService()
	registryService := opcache.NewService()
	routerService := router.New(reqx.)

	sm.CredentialsService = credentialsService
	sm.PreferencesService = preferencesService
	sm.OperationsService = registryService
	sm.FacadeService = facade.NewService(preferencesService, loaderService, trackerService, registryService, routerService, sm.sources())

	return sm
}

func (sm *ServiceManager) InitConsoleLogging() {
	zerolog.ErrorMarshalFunc = trace.ErrorMarshalFunc
	log.Logger = log.Logger.Output(trace.ConsoleTraceWriter())
}

func (sm *ServiceManager) sources() []source.ID {
	return []source.ID{
		source.CharacterTavern,
		source.ChubAI,
		source.NyaiMe,
		source.PepHop,
		source.Pygmalion,
		source.WyvernChat,
	}
}

func storeProvider(
	client *ent.Client,
	vault ivault.Vault,
	opts opts.PngOptions,
) istore.Service {
	return store.NewService(client, vault.Name, vault.CardsDir, opts)
}
