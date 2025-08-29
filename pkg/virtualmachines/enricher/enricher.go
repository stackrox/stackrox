package enricher

import (
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
)

// VirtualMachineEnricher provides functions for enriching VMs with vulnerability data.
//
//go:generate mockgen-wrapper
type VirtualMachineEnricher interface {
	EnrichVirtualMachineWithVulnerabilities(vm *storage.VirtualMachine, indexReport *v4.IndexReport) error
}
