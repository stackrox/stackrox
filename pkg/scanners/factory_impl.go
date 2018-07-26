package scanners

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/scanners/types"
)

type factoryImpl struct {
	creators map[string]Creator
}

func (e *factoryImpl) CreateScanner(source *v1.ImageIntegration) (types.ImageScanner, error) {
	creator, exists := e.creators[source.GetType()]
	if !exists {
		return nil, fmt.Errorf("Scanner with type '%s' does not exist", source.GetType())
	}
	return creator(source)
}
