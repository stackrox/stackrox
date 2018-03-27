package registries

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// Creator is a func stub that defines how to create an image registry
type Creator func(integration *v1.ImageIntegration) (ImageRegistry, error)

// Registry maps the registry name to it's creation function
var Registry = map[string]Creator{}

// CreateRegistry checks to make sure the integration exists and then tries to generate a new Registry
// returns an error if the creation was unsuccessful
func CreateRegistry(source *v1.ImageIntegration) (ImageRegistry, error) {
	creator, exists := Registry[source.GetType()]
	if !exists {
		return nil, fmt.Errorf("Registry with type '%s' does not exist", source.GetType())
	}
	return creator(source)
}
