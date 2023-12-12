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
func (e *setImpl) UpdateImageIntegration(integration *storage.ImageIntegration) error {
	if err := validateCommonFields(integration); err != nil {
		return err
	}

	var isRegistry bool
	var isScanner bool
	errorList := errorhelpers.NewErrorList("updating integration")
	for _, category := range integration.GetCategories() {
		switch category {
		case storage.ImageIntegrationCategory_REGISTRY:
			isRegistry = true
			if err := e.registrySet.UpdateImageIntegration(integration); err != nil {
				errorList.AddError(err)
			}
		case storage.ImageIntegrationCategory_SCANNER:
			isScanner = true
			if err := e.scannerSet.UpdateImageIntegration(integration); err != nil {
				errorList.AddError(err)
			}
		case storage.ImageIntegrationCategory_NODE_SCANNER: // This is because node scanners are implemented into image integrations
		default:
			errorList.AddError(fmt.Errorf("source category %q has not been implemented", category))
		}
	}

	// An integration may have a category removed, for example, if an integration went from being
	// both a registry + scanner to just a registry. On update we need to remove the integration
	// from the sets it should no longer be a part of.
	if !isRegistry {
		if err := e.registrySet.RemoveImageIntegration(integration.GetId()); err != nil {
			log.Warnf("Unable to remove integration %q (%s) from registry set: %v", integration.GetName(), integration.GetId(), err)
		}
	}

	if !isScanner {
		if err := e.scannerSet.RemoveImageIntegration(integration.GetId()); err != nil {
			log.Warnf("Unable to remove integration %q (%s) from scanner set: %v", integration.GetName(), integration.GetId(), err)
		}
	}

	rErr := e.reporter.Register(integration.GetId(), integration.GetName(), storage.IntegrationHealth_IMAGE_INTEGRATION)
	if rErr != nil {
		log.Errorf("Error registering health for integration %s: %s", integration.GetId(), integration.GetName())
	}

	return errorList.ToError()
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
