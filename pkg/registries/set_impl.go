package registries

import (
	"sync"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/registries/types"
)

type setImpl struct {
	lock sync.RWMutex

	factory      Factory
	integrations map[string]types.ImageRegistry
}

// GetAll returns the set of integrations that are active.
func (e *setImpl) GetAll() []types.ImageRegistry {
	e.lock.RLock()
	defer e.lock.RUnlock()

	integrations := make([]types.ImageRegistry, 0, len(e.integrations))
	for _, i := range e.integrations {
		integrations = append(integrations, i)
	}
	return integrations
}

// GetRegistryMetadataByImage returns the config for a registry that contains the input image.
func (e *setImpl) GetRegistryMetadataByImage(image *v1.Image) *types.Config {
	e.lock.RLock()
	defer e.lock.RUnlock()

	for _, i := range e.integrations {
		if i.Match(image) {
			return i.Config()
		}
	}
	return nil
}

// Match returns whether a registry in the set has the given image.
func (e *setImpl) Match(image *v1.Image) bool {
	e.lock.RLock()
	defer e.lock.RUnlock()

	for _, i := range e.integrations {
		if i.Match(image) {
			return true
		}
	}
	return false
}

// Clear removes all present integrations.
func (e *setImpl) Clear() {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.integrations = nil
}

// UpdateImageIntegration updates the integration with the matching id to a new configuration.
func (e *setImpl) UpdateImageIntegration(integration *v1.ImageIntegration) (err error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.integrations[integration.GetId()], err = e.factory.CreateRegistry(integration)
	return
}

// RemoveImageIntegration removes the integration with a matching id if one exists.
func (e *setImpl) RemoveImageIntegration(id string) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	delete(e.integrations, id)
	return nil
}
