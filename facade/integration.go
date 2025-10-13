package facade

import (
	"github.com/r3dpixel/card-fetcher/router"
	"github.com/r3dpixel/card-fetcher/source"
)

func (f *Facade) GetSourceStatuses() map[source.ID]router.IntegrationStatus {
	return f.router.CheckIntegrations()
}

func (f *Facade) GetSources() []source.ID {
	return f.router.Sources()
}
