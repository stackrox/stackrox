package enricher

import (
	"context"

	"github.com/pkg/errors"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scanners/types"
)

var (
	log = logging.LoggerForModule()
)

type enricherImpl struct {
	vmScanner types.VirtualMachineScanner
}

func New(scanner types.VirtualMachineScanner) VirtualMachineEnricher {
	return &enricherImpl{
		vmScanner: scanner,
	}
}

func (e *enricherImpl) EnrichVirtualMachineWithVulnerabilities(vm *storage.VirtualMachine, indexReport *v4.IndexReport) error {
	// Clear any pre-existing notes
	vm.Notes = vm.GetNotes()[:0]

	if e.vmScanner == nil {
		vm.Notes = append(vm.Notes, storage.VirtualMachine_MISSING_SCANNER)
		return errors.New("Scanner V4 client not available for VM enrichment")
	}

	sema := e.vmScanner.MaxConcurrentNodeScanSemaphore()
	_ = sema.Acquire(context.Background(), 1)
	defer sema.Release(1)

	scan, err := e.vmScanner.GetVirtualMachineScan(vm, indexReport)
	if err != nil {
		vm.Notes = append(vm.Notes, storage.VirtualMachine_SCAN_FAILED)
		return errors.Wrap(err, "getting scan for VM")
	}

	vm.Scan = scan
	log.Debugf("Enriched VM %s with %d components", vm.GetName(), len(vm.GetScan().GetComponents()))
	return nil
}
