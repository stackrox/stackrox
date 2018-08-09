package registries

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/registries/types"
)

// Set provides an interface for reading the active set of image integrations.
type Set interface {
	GetAll() []types.ImageRegistry
	Match(image *v1.Image) bool
	GetRegistryMetadataByImage(image *v1.Image) *types.Config

	Clear()
	UpdateImageIntegration(integration *v1.ImageIntegration) error
	RemoveImageIntegration(id string) error
}

// NewSet returns a new Set instance.
func NewSet(factory Factory) Set {
	return &setImpl{
		factory:      factory,
		integrations: make(map[string]types.ImageRegistry),
	}
}
