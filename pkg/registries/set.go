package registries

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
)

// Set provides an interface for reading the active set of image integrations.
//
//go:generate mockgen-wrapper
type Set interface {
	GetAll() []types.ImageRegistry
	Match(image *storage.ImageName) bool
	GetRegistryMetadataByImage(image *storage.Image) *types.Config
	GetRegistryByImage(image *storage.Image) types.Registry

	IsEmpty() bool
	Clear()
	UpdateImageIntegration(integration *storage.ImageIntegration) error
	RemoveImageIntegration(id string) error
}

// NewSet returns a new Set instance.
func NewSet(factory Factory) Set {
	return &setImpl{
		factory:      factory,
		integrations: make(map[string]types.ImageRegistry),
	}
}
