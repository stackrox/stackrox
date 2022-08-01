package registries

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
)

type factoryImpl struct {
	creators map[string]Creator
}

type registryWithDataSource struct {
	types.Registry
	source *storage.ImageIntegration
}

func (r *registryWithDataSource) Source() *storage.ImageIntegration {
	return r.source
}

func (e *factoryImpl) CreateRegistry(source *storage.ImageIntegration) (types.ImageRegistry, error) {
	creator, exists := e.creators[source.GetType()]
	if !exists {
		return nil, fmt.Errorf("registry with type '%s' does not exist", source.GetType())
	}
	integration, err := creator(source)
	if err != nil {
		return nil, err
	}

	return &registryWithDataSource{
		Registry: integration,
		source:   source,
	}, nil
}
