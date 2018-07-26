package scanners

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/scanners/types"
)

// Set provides an interface for reading the active set of image integrations.
type Set interface {
	GetAll() []types.ImageScanner

	Clear()
	UpdateImageIntegration(integration *v1.ImageIntegration) error
	RemoveImageIntegration(id string) error
}

// NewSet returns a new Set instance.
func NewSet(factory Factory) Set {
	return &setImpl{
		factory:      factory,
		integrations: make(map[string]types.ImageScanner),
	}
}
