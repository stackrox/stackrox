package integration

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/scanners"
)

// Set provides an interface for reading the active set of image integrations.
//
//go:generate mockgen-wrapper
type Set interface {
	RegistryFactory() registries.Factory
	ScannerFactory() scanners.Factory

	RegistrySet() registries.Set
	ScannerSet() scanners.Set

	Clear()
	UpdateImageIntegration(integration *storage.ImageIntegration) error
	RemoveImageIntegration(id string) error
}

// NewSet returns a new Set instance.
func NewSet(reporter integrationhealth.Reporter) Set {
	registryFactory := registries.NewFactory(registries.FactoryOptions{
		CreatorFuncsWithoutRepoList: registries.AllCreatorFuncsWithoutRepoList,
	})

	registrySet := registries.NewSet(registryFactory)

	scannerFactory := scanners.NewFactory(registrySet)
	scannerSet := scanners.NewSet(scannerFactory)

	return &setImpl{
		registryFactory: registryFactory,
		scannerFactory:  scannerFactory,

		registrySet: registrySet,
		scannerSet:  scannerSet,
		reporter:    reporter,
	}
}
