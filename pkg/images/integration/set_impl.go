package integration

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/scanners"
)

var (
	log = logging.LoggerForModule()
)

type setImpl struct {
	registryFactory registries.Factory
	scannerFactory  scanners.Factory

	registrySet registries.Set
	scannerSet  scanners.Set

	reporter integrationhealth.Reporter
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
func (e *setImpl) UpdateImageIntegration(integration *storage.ImageIntegration) (err error) {
	err = validateCommonFields(integration)
	if err != nil {
		return
	}

	var isRegistry bool
	var isScanner bool
	for _, category := range integration.GetCategories() {
		switch category {
		case storage.ImageIntegrationCategory_REGISTRY:
			isRegistry = true
			err = e.registrySet.UpdateImageIntegration(integration)
		case storage.ImageIntegrationCategory_SCANNER:
			isScanner = true
			err = e.scannerSet.UpdateImageIntegration(integration)
		case storage.ImageIntegrationCategory_NODE_SCANNER: // This is because node scanners are implemented into image integrations
		default:
			err = fmt.Errorf("source category %q has not been implemented", category)
		}
	}

	// An integration may have a category removed, for example, if an integration went from being
	// both a registry + scanner to just a registry. On update we need to remove the integration
	// from the sets it should no longer be a part of.
	if !isRegistry {
		e.registrySet.RemoveImageIntegration(integration.GetId())
	}

	if !isScanner {
		e.scannerSet.RemoveImageIntegration(integration.GetId())
	}

	rErr := e.reporter.Register(integration.GetId(), integration.GetName(), storage.IntegrationHealth_IMAGE_INTEGRATION)
	if rErr != nil {
		log.Errorf("Error registering health for integration %s: %s", integration.GetId(), integration.GetName())
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
	if err != nil {
		return
	}

	err = e.reporter.RemoveIntegrationHealth(id)
	return
}

func validateCommonFields(source *storage.ImageIntegration) error {
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
