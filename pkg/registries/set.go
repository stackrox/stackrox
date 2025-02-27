package registries

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
)

// Set provides an interface for reading the active set of image integrations.
//
//go:generate mockgen-wrapper
type Set interface {
	GetAll() []types.ImageRegistry
	GetAllUnique() []types.ImageRegistry
	Match(image *storage.ImageName) bool
	GetRegistryMetadataByImage(ctx context.Context, image *storage.Image) *types.Config
	GetRegistryByImage(image *storage.Image) types.Registry

	IsEmpty() bool
	Len() int
	Clear()
	UpdateImageIntegration(integration *storage.ImageIntegration) (bool, error)
	RemoveImageIntegration(id string) error
}

// NewSet returns a new Set instance.
func NewSet(factory Factory, creatorOpts ...types.CreatorOption) Set {
	return &setImpl{
		factory:      factory,
		integrations: make(map[string]types.ImageRegistry),
		creatorOpts:  creatorOpts,
	}
}
