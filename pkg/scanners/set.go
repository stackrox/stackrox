package scanners

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scanners/types"
)

// Set provides an interface for reading the active set of image integrations.
//
//go:generate mockgen-wrapper
type Set interface {
	GetAll() []types.ImageScannerWithDataSource

	IsEmpty() bool
	Clear()
	UpdateImageIntegration(integration *storage.ImageIntegration) error
	RemoveImageIntegration(id string) error
}

// NewSet returns a new Set instance.
func NewSet(factory Factory) Set {
	return &setImpl{
		factory:      factory,
		integrations: make(map[string]types.ImageScannerWithDataSource),
	}
}
