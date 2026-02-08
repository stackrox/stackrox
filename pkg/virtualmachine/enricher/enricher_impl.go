package enricher

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	pkgScanners "github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

type enricherImpl struct {
	scanners map[string]types.VirtualMachineScanner
	creators map[string]pkgScanners.VirtualMachineScannerCreator

	lock sync.RWMutex
}

func newWithCreator(fn func() (string, func(*storage.ImageIntegration) (types.VirtualMachineScanner, error))) VirtualMachineEnricher {
	enricher := &enricherImpl{
		scanners: make(map[string]types.VirtualMachineScanner),
		creators: make(map[string]pkgScanners.VirtualMachineScannerCreator),
	}
	name, creator := fn()
	enricher.creators[name] = creator
	return enricher
}

func (e *enricherImpl) EnrichVirtualMachineWithVulnerabilities(vm *storage.VirtualMachine, indexReport *v4.IndexReport) error {
	// Clear any pre-existing notes
	vm.Notes = vm.GetNotes()[:0]

	scanners := e.getScanners()
	if len(scanners) == 0 {
		vm.Notes = append(vm.Notes, storage.VirtualMachine_MISSING_SCAN_DATA)
		return errors.New("no scanner V4 client available for virtual machine enrichment")
	}

	errorList := errorhelpers.NewErrorList(fmt.Sprintf("error scanning virtual machine %s:%s", vm.GetClusterName(), vm.GetName()))
	for _, scanner := range scanners {
		if err := enrichVirtualMachineWithScanner(vm, indexReport, scanner); err != nil {
			errorList.AddError(err)
		}
	}
	return errorList.ToError()
}

func (e *enricherImpl) getScanners() []types.VirtualMachineScanner {
	e.lock.RLock()
	defer e.lock.RUnlock()

	res := make([]types.VirtualMachineScanner, 0, len(e.scanners))
	for _, scanner := range e.scanners {
		res = append(res, scanner)
	}
	return res
}

func enrichVirtualMachineWithScanner(machine *storage.VirtualMachine, indexReport *v4.IndexReport, scanner types.VirtualMachineScanner) error {
	if scanner == nil {
		machine.Notes = append(machine.Notes, storage.VirtualMachine_MISSING_SCAN_DATA)
		return errors.New("scanner v4 client not available for virtual machine enrichment")
	}
	sema := scanner.MaxConcurrentNodeScanSemaphore()
	_ = sema.Acquire(context.Background(), 1)
	defer sema.Release(1)

	scan, err := scanner.GetVirtualMachineScan(machine, indexReport)
	if err != nil {
		// Currently, the error paths in GetVirtualMachineScan all come from missing data
		// to perform an actual scan.
		// The function signature could be changed to return the note to be added to the
		// virtual machine notes
		machine.Notes = append(machine.Notes, storage.VirtualMachine_MISSING_SCAN_DATA)
		return errors.Wrap(err, "getting scan for VM")
	}
	machine.Scan = scan
	log.Debugf("Enriched VM %s with %d components", machine.GetName(), len(machine.GetScan().GetComponents()))
	return nil
}

func (e *enricherImpl) UpsertVirtualMachineIntegration(integration *storage.ImageIntegration) error {
	vmScanner, err := e.createVirtualMachineScanner(integration)
	if err != nil {
		return errors.Wrap(err, "adding or updating integration")
	}

	e.lock.Lock()
	defer e.lock.Unlock()

	e.scanners[integration.GetId()] = vmScanner
	return nil
}

func (e *enricherImpl) createVirtualMachineScanner(integration *storage.ImageIntegration) (types.VirtualMachineScanner, error) {
	return e.getScannerCreator(integration)(integration)
}

func (e *enricherImpl) getScannerCreator(integration *storage.ImageIntegration) func(*storage.ImageIntegration) (types.VirtualMachineScanner, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()

	return e.creators[integration.GetType()]
}

func (e *enricherImpl) RemoveVirtualMachineIntegration(id string) {
	e.lock.Lock()
	defer e.lock.Unlock()

	delete(e.scanners, id)
}
