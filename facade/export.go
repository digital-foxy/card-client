package facade

import (
	"sync"

	"github.com/digital-foxy/card-client/operation"
	"github.com/digital-foxy/card-client/preferences"
	"github.com/digital-foxy/card-client/store/templater"
	"github.com/digital-foxy/card-client/tracker"
)

// exportService handles card export operations
type exportService struct {
	vault       *vaultManager
	tracker     tracker.Service
	preferences preferences.Service
	registry    operation.Registry
	cache       *cacheManager
	workers     int

	templateMu       sync.Mutex
	template         string
	templater        *templater.Templater
	compiledTemplate *templater.CompiledTemplate
}

func newExportService(
	vault *vaultManager,
	tracker tracker.Service,
	pref preferences.Service,
	registry operation.Registry,
	cache *cacheManager,
	workers int,
) *exportService {
	template := pref.GetString(preferences.ExportTemplateKey)
	templater := templater.New()
	return &exportService{
		vault:            vault,
		tracker:          tracker,
		preferences:      pref,
		registry:         registry,
		cache:            cache,
		workers:          workers,
		template:         template,
		templater:        templater,
		compiledTemplate: templater.Compile(template),
	}
}

// getCompiledTemplate returns the current compiled template, refreshing if needed.
func (s *exportService) getCompiledTemplate() *templater.CompiledTemplate {
	newTemplate := s.preferences.GetString(preferences.ExportTemplateKey)

	s.templateMu.Lock()
	defer s.templateMu.Unlock()

	if s.template != newTemplate {
		s.template = newTemplate
		s.compiledTemplate = s.templater.Compile(s.template)
	}
	return s.compiledTemplate
}
