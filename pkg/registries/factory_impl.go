package registries

import (
	"fmt"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/registries/types"
)

type factoryImpl struct {
	creators map[string]Creator
}

type registryWithDataSource struct {
	types.Registry
	datasource *storage.DataSource
}

func (r *registryWithDataSource) DataSource() *storage.DataSource {
	return r.datasource
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
		datasource: &storage.DataSource{
			Id:   source.GetId(),
			Name: source.GetName(),
		},
	}, nil
}
