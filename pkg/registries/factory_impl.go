package registries

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
)

type factoryImpl struct {
	creators map[string]Creator
}

func (e *factoryImpl) CreateRegistry(source *storage.ImageIntegration) (types.ImageRegistry, error) {
	creator, exists := e.creators[source.GetType()]
	if !exists {
		return nil, fmt.Errorf("Registry with type '%s' does not exist", source.GetType())
	}
	integration, err := creator(source)
	if err != nil {
		return nil, err
	}
	if err := integration.Test(); err != nil {
		return nil, fmt.Errorf("Test on integration failed: %s", err)
	}
	return integration, nil
}
