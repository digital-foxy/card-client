package facade

import (
	"github.com/r3dpixel/card-fetcher/router"
	"github.com/r3dpixel/card-fetcher/source"
)

func (s *Service) GetSourceStatuses() map[source.ID]router.IntegrationStatus {
	return s.routerService.CheckIntegrations()
}

func (s *Service) GetSources() []source.ID {
	return s.routerService.Sources()
}
