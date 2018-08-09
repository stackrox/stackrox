package integration

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/scanners"
)

// Set provides an interface for reading the active set of image integrations.
type Set interface {
	RegistryFactory() registries.Factory
	ScannerFactory() scanners.Factory

	RegistrySet() registries.Set
	ScannerSet() scanners.Set

	Clear()
	UpdateImageIntegration(integration *v1.ImageIntegration) error
	RemoveImageIntegration(id string) error
}

// NewSet returns a new Set instance.
func NewSet() Set {
	registryFactory := registries.NewFactory()
	registrySet := registries.NewSet(registryFactory)

	scannerFactory := scanners.NewFactory(registrySet)
	scannerSet := scanners.NewSet(scannerFactory)

	return &setImpl{
		registryFactory: registryFactory,
		scannerFactory:  scannerFactory,

		registrySet: registrySet,
		scannerSet:  scannerSet,
	}
}
