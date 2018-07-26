package registries

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/registries/types"
)

type factoryImpl struct {
	creators map[string]Creator
}

func (e *factoryImpl) CreateRegistry(source *v1.ImageIntegration) (types.ImageRegistry, error) {
	creator, exists := e.creators[source.GetType()]
	if !exists {
		return nil, fmt.Errorf("Registry with type '%s' does not exist", source.GetType())
	}
	return creator(source)
}
