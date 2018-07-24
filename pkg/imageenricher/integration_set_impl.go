package imageenricher

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/sources"
)

type integrationSetImpl struct {
	integrations map[string]*sources.ImageIntegration
	lock         sync.RWMutex
}

// UpdateImageIntegration updates the enricher's map of active image integratinos
func (e *integrationSetImpl) UpdateImageIntegration(integration *sources.ImageIntegration) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.integrations[integration.GetId()] = integration
}

// RemoveImageIntegration removes a image integration from the enricher's map of active image integrations
func (e *integrationSetImpl) RemoveImageIntegration(id string) {
	e.lock.Lock()
	defer e.lock.Unlock()

	delete(e.integrations, id)
}

func (e *integrationSetImpl) GetRegistryMetadataByImage(image *v1.Image) *registries.Config {
	e.lock.RLock()
	defer e.lock.RUnlock()

	for _, i := range e.integrations {
		if i.Registry != nil && i.Registry.Match(image) {
			return i.Registry.Config()
		}
	}
	return nil
}

// Match determines if an image integration matches
func (e *integrationSetImpl) Match(image *v1.Image) bool {
	e.lock.RLock()
	defer e.lock.RUnlock()

	for _, i := range e.integrations {
		if i.Registry != nil && i.Registry.Match(image) {
			return true
		}
	}
	return false
}

// Match determines if an image integration matches
func (e *integrationSetImpl) GetAll() []*sources.ImageIntegration {
	e.lock.RLock()
	defer e.lock.RUnlock()

	integrations := make([]*sources.ImageIntegration, 0, len(e.integrations))
	for _, i := range e.integrations {
		integrations = append(integrations, i)
	}
	return integrations
}
