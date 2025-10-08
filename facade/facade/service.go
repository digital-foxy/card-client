package facade

import (
	"sync"

	"github.com/r3dpixel/card-fetcher/router"
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/toolkit/trace"
	"github.com/rs/zerolog/log"
)

type Service struct {
	builderMutex    sync.Mutex
	fileNameBuilder FileNameBuilder

	updateRequestCache *cache.RequestCache
	exportRequestCache *cache.RequestCache

	pref           preferences.Service
	loaderService  loader.Service
	trackerService tracker.Service
	registry       operation.Registry
	routerService  *router.Router

	storeService store.Service
	storeMutex   sync.RWMutex
}

func NewService(
	pref preferences.Service,
	loader loader.Service,
	tracker tracker.Service,
	registry operation.Registry,
	router *router.Router,
	sources []source.ID,
) *Service {
	router.RegisterFetchers(sources...)
	return &Service{
		updateRequestCache: cache.NewRequestCache(0),
		exportRequestCache: cache.NewRequestCache(0),
		pref:               pref,
		loaderService:      loader,
		trackerService:     tracker,
		registry:           registry,
		routerService:      router,
	}
}

func (s *Service) SetFileNameBuilder(builder FileNameBuilder) {
	s.builderMutex.Lock()
	defer s.builderMutex.Unlock()

	s.fileNameBuilder = builder
}

func (s *Service) LoadVault(vault string) error {
	if !s.storeMutex.TryLock() {
		return trace.Err().Msg("Vault is busy")
	}
	defer s.storeMutex.Unlock()

	newStore, err := s.loaderService.LoadVault(vault)
	if err != nil {
		log.Error().Err(err).
			Str(trace.SERVICE, "facade").
			Str(trace.ACTIVITY, "load-vault").
			Str("vault", vault).
			Msg("Could not load vault")
		return trace.Err().Msg("Could not load vault")
	}

	oldStore := s.storeService
	s.storeService = newStore

	if oldStore != nil {
		err = oldStore.Close()
		log.Warn().Err(err).
			Str(trace.SERVICE, "facade").
			Str(trace.ACTIVITY, "load-vault").
			Str("vault", vault).
			Msg("Could not close old store service")
	}

	return nil
}

func (s *Service) UnloadVault() error {
	if !s.storeMutex.TryLock() {
		return trace.Err().Msg("Vault is busy")
	}
	defer s.storeMutex.Unlock()

	if s.storeService != nil {
		err := s.storeService.Close()
		log.Warn().Err(err).
			Str(trace.SERVICE, "facade").
			Str(trace.ACTIVITY, "unload-vault").
			Msg("Could not unload vault")
	}

	s.storeService = nil
	return nil
}

func (s *Service) beginReadStoreOp() (unlock func(), err error) {
	if !s.storeMutex.TryRLock() {
		return nil, trace.Err().Msg("Vault is loading")
	}

	if s.storeService == nil {
		s.storeMutex.RUnlock()
		return nil, trace.Err().Msg("No vault loaded")
	}

	return s.storeMutex.RUnlock, nil
}
