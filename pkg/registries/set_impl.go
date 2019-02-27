package registries

import (
	"sort"
	"sync"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
)

type setImpl struct {
	lock sync.RWMutex

	factory      Factory
	integrations map[string]types.ImageRegistry
}

func (e *setImpl) getSortedRegistriesNoLock() []types.ImageRegistry {
	integrations := make([]types.ImageRegistry, 0, len(e.integrations))
	for _, i := range e.integrations {
		integrations = append(integrations, i)
	}
	// This just ensures that the registries that have username/passwords are processed first
	sort.SliceStable(integrations, func(i, j int) bool {
		return integrations[i].Config().Username != "" && integrations[j].Config().Username == ""
	})
	return integrations
}

// GetAll returns the set of integrations that are active.
func (e *setImpl) GetAll() []types.ImageRegistry {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.getSortedRegistriesNoLock()
}

// GetRegistryMetadataByImage returns the config for a registry that contains the input image.
func (e *setImpl) GetRegistryMetadataByImage(image *storage.Image) *types.Config {
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
func (e *setImpl) Match(image *storage.Image) bool {
	e.lock.RLock()
	defer e.lock.RUnlock()

	integrations := e.getSortedRegistriesNoLock()
	for _, i := range integrations {
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

	e.integrations = make(map[string]types.ImageRegistry)
}

// UpdateImageIntegration updates the integration with the matching id to a new configuration.
func (e *setImpl) UpdateImageIntegration(integration *storage.ImageIntegration) error {
	i, err := e.factory.CreateRegistry(integration)
	if err != nil {
		return err
	}
	e.lock.Lock()
	defer e.lock.Unlock()

	e.integrations[integration.GetId()] = i
	return nil
}

// RemoveImageIntegration removes the integration with a matching id if one exists.
func (e *setImpl) RemoveImageIntegration(id string) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	delete(e.integrations, id)
	return nil
}
