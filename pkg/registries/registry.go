package registries

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

// Creator is a func stub that defines how to create an image registry
type Creator func(registry *v1.Registry) (ImageRegistry, error)

// Registry maps the registry name to it's creation function
var Registry = map[string]Creator{}

// CreateRegistry checks to make sure the integration exists and then tries to generate a new Registry
// returns an error if the creation was unsuccessful
func CreateRegistry(registry *v1.Registry) (ImageRegistry, error) {
	creator, exists := Registry[registry.Type]
	if !exists {
		return nil, fmt.Errorf("Registry with type %v does not exist", registry.Type)
	}
	return creator(registry)
}
