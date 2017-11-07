package registries

import "bitbucket.org/stack-rox/apollo/apollo/registries/types"

// Creator is a func stub that defines how to create an image registry
type Creator func(endpoint string, config map[string]string) (types.ImageRegistry, error)

// Registry maps the registry name to it's creation function
var Registry = map[string]Creator{}
