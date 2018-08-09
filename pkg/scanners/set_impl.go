package scanners

import (
	"sync"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/scanners/types"
)

type setImpl struct {
	lock sync.RWMutex

	factory      Factory
	integrations map[string]types.ImageScanner
}

// GetAll returns the set of integrations that are active.
func (e *setImpl) GetAll() []types.ImageScanner {
	e.lock.RLock()
	defer e.lock.RUnlock()

	integrations := make([]types.ImageScanner, 0, len(e.integrations))
	for _, i := range e.integrations {
		integrations = append(integrations, i)
	}
	return integrations
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

	e.integrations[integration.GetId()], err = e.factory.CreateScanner(integration)
	return
}

// RemoveImageIntegration removes the integration with a matching id if one exists.
func (e *setImpl) RemoveImageIntegration(id string) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	delete(e.integrations, id)
	return nil
}
