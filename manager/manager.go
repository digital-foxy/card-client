package manager

import (
	"github.com/r3dpixel/card-client/internal/credentials"
	"github.com/r3dpixel/card-client/internal/ent"
	"github.com/r3dpixel/card-client/internal/facade"
	ifilter "github.com/r3dpixel/card-client/internal/filter"
	"github.com/r3dpixel/card-client/internal/loader"
	"github.com/r3dpixel/card-client/internal/operation"
	"github.com/r3dpixel/card-client/internal/preferences"
	"github.com/r3dpixel/card-client/internal/store"
	"github.com/r3dpixel/card-client/internal/tracker"
	"github.com/r3dpixel/card-client/internal/vault"
	"github.com/r3dpixel/card-client/opts"
	icredentials "github.com/r3dpixel/card-client/services/credentials"
	ifacade "github.com/r3dpixel/card-client/services/facade"
	"github.com/r3dpixel/card-client/services/filter"
	ioperation "github.com/r3dpixel/card-client/services/operation"
	ipreferences "github.com/r3dpixel/card-client/services/preferences"
	istore "github.com/r3dpixel/card-client/services/store"
	ivault "github.com/r3dpixel/card-client/services/vault"
	"github.com/r3dpixel/card-fetcher/factory"
	"github.com/r3dpixel/card-fetcher/router"
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/toolkit/trace"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ServiceManager struct {
	PreferencesService ipreferences.Service
	CredentialsService icredentials.Service
	OperationsService  ioperation.Service
	VaultService       ivault.Service
	FilterService      filter.Service
	FacadeService      ifacade.Service
}

func NewManager(config opts.AppConfig) *ServiceManager {
	sm := &ServiceManager{}
	vaultService := vault.NewService(config.VaultOptions)

	credentialsService := credentials.NewService()
	preferencesService := preferences.NewService(config.PreferencesOptions)
	loaderService := loader.NewService(config.StoreOptions, vaultService, storeProvider)
	trackerService := tracker.NewService()
	registryService := operation.NewRegistry(operation.DefaultIdGenerator())
	routerService := router.New(router.Options{
		FactoryOptions: factory.FactoryOptions{
			PygmalionIdentityProvider: credentialsService.GetReader(icredentials.Pygmalion),
		},
		ClientOptions: config.ClientOptions,
	})

	sm.CredentialsService = credentialsService
	sm.PreferencesService = preferencesService
	sm.OperationsService = registryService
	sm.VaultService = vaultService
	sm.FilterService = &ifilter.Service{}
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
