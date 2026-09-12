package enricher

import (
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
)

// VirtualMachineEnricher provides functions for enriching VMs with vulnerability data.
// It also owns the explicit VM-scanner integration lifecycle so callers can
// keep one shared enricher instance in sync with Central integration state.
//
//go:generate mockgen-wrapper
type VirtualMachineEnricher interface {
	// EnrichVirtualMachineWithVulnerabilities enriches the given VM using the
	// currently selected VM scanner.
	EnrichVirtualMachineWithVulnerabilities(vm *storage.VirtualMachine, indexReport *v4.IndexReport) error
	// UpsertVirtualMachineIntegration creates or replaces the explicit VM scanner
	// associated with the provided image integration.
	UpsertVirtualMachineIntegration(integration *storage.ImageIntegration) error
	// RemoveVirtualMachineIntegration removes the explicit VM scanner associated
	// with the provided integration ID.
	RemoveVirtualMachineIntegration(id string)
}
