package integration

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/scanners"
)

type setImpl struct {
	registryFactory registries.Factory
	scannerFactory  scanners.Factory

	registrySet registries.Set
	scannerSet  scanners.Set
}

func (e *setImpl) RegistryFactory() registries.Factory {
	return e.registryFactory
}

func (e *setImpl) ScannerFactory() scanners.Factory {
	return e.scannerFactory
}

// RegistrySet returns the registries.Set holding all active registry integrations.
func (e *setImpl) RegistrySet() registries.Set {
	return e.registrySet
}

// ScannerSet returns the scanners.Set holding all active scanner integrations.
func (e *setImpl) ScannerSet() scanners.Set {
	return e.scannerSet
}

// Clear removes all present integrations.
func (e *setImpl) Clear() {
	e.registrySet.Clear()
	e.scannerSet.Clear()
}

// UpdateImageIntegration updates the integration with the matching id to a new configuration.
func (e *setImpl) UpdateImageIntegration(integration *v1.ImageIntegration) (err error) {
	err = validateCommonFields(integration)
	if err != nil {
		return
	}

	for _, category := range integration.GetCategories() {
		switch category {
		case v1.ImageIntegrationCategory_REGISTRY:
			err = e.registrySet.UpdateImageIntegration(integration)
		case v1.ImageIntegrationCategory_SCANNER:
			err = e.scannerSet.UpdateImageIntegration(integration)
		default:
			err = fmt.Errorf("Source category '%s' has not been implemented", category)
		}
	}
	return
}

// RemoveImageIntegration removes the integration with a matching id if one exists.
func (e *setImpl) RemoveImageIntegration(id string) (err error) {
	err = e.registrySet.RemoveImageIntegration(id)
	if err != nil {
		return
	}
	err = e.scannerSet.RemoveImageIntegration(id)
	return
}

func validateCommonFields(source *v1.ImageIntegration) error {
	errorList := errorhelpers.NewErrorList("Validation")
	if source.GetName() == "" {
		errorList.AddString("Source name must be defined")
	}
	if source.GetType() == "" {
		errorList.AddString("Source type must be defined")
	}
	if len(source.GetCategories()) == 0 {
		errorList.AddString("At least one category must be defined")
	}
	return errorList.ToError()
}
