package integration

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/scanners"
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
