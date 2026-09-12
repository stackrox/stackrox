package enricher

import (
	"context"
	"maps"
	"slices"

	"github.com/pkg/errors"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/scannerv4"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

type enricherImpl struct {
	// resolveScanner preserves the pre-category fallback path so existing
	// deployments can still use the active Scanner V4 integration until an
	// explicit VM-scanner category is configured.
	resolveScanner func() types.VirtualMachineScanner
	// creators is keyed by integration type because VM-scanner support is added
	// per scanner implementation, not every image integration type.
	creators map[string]scanners.VirtualMachineScannerCreator
	// scanners stores explicitly configured VM scanners keyed by integration ID.
	// We keep all configured scanners so selection can remain deterministic.
	scanners map[string]types.VirtualMachineScanner
	// lock protects both creators and explicit scanners because they are updated
	// from integration lifecycle events while reads happen during enrichment.
	lock sync.RWMutex
}

// New returns a VM enricher that uses explicit VM-scanner integrations when
// present and otherwise falls back to the provided legacy resolver.
func New(resolveScanner func() types.VirtualMachineScanner) VirtualMachineEnricher {
	name, creator := scannerv4.VirtualMachineScannerCreator()
	return newWithCreator(resolveScanner, creatorRegistration{name: name, creator: creator})
}

type creatorRegistration struct {
	name    string
	creator scanners.VirtualMachineScannerCreator
}

// newWithCreator wires the explicit VM-scanner creators used by tests and
// production code. The fallback resolver remains injectable so the current
// integration-set behavior can coexist with the new category model.
func newWithCreator(
	resolveScanner func() types.VirtualMachineScanner,
	registrations ...creatorRegistration,
) VirtualMachineEnricher {
	enricher := &enricherImpl{
		resolveScanner: resolveScanner,
		creators:       make(map[string]scanners.VirtualMachineScannerCreator, len(registrations)),
		scanners:       make(map[string]types.VirtualMachineScanner),
	}
	for _, registration := range registrations {
		enricher.creators[registration.name] = registration.creator
	}
	return enricher
}

func (e *enricherImpl) EnrichVirtualMachineWithVulnerabilities(vm *storage.VirtualMachine, indexReport *v4.IndexReport) error {
	// Clear any pre-existing notes
	vm.Notes = vm.GetNotes()[:0]

	vmScanner := e.resolveActiveScanner()
	if vmScanner == nil {
		vm.Notes = append(vm.Notes, storage.VirtualMachine_MISSING_SCANNER)
		return errors.New("Scanner V4 client not available for VM enrichment")
	}

	sema := vmScanner.MaxConcurrentNodeScanSemaphore()
	_ = sema.Acquire(context.Background(), 1)
	defer sema.Release(1)

	scan, err := vmScanner.GetVirtualMachineScan(vm, indexReport)
	if err != nil {
		vm.Notes = append(vm.Notes, storage.VirtualMachine_SCAN_FAILED)
		return errors.Wrap(err, "getting scan for VM")
	}

	vm.Scan = scan
	log.Debugf("Enriched VM %s with %d components", vm.GetName(), len(vm.GetScan().GetComponents()))
	return nil
}

// resolveActiveScanner returns the active VM scanner.
// If explicit VM-scanner integrations exist, it selects the first one in the
// deterministic order produced by explicitVMScanners. Otherwise it falls back
// to the legacy resolver so clusters without the VM-scanner category keep their
// existing Scanner V4 behavior.
func (e *enricherImpl) resolveActiveScanner() types.VirtualMachineScanner {
	explicit := e.explicitVMScanners()
	if len(explicit) > 0 {
		return explicit[0]
	}
	if e.resolveScanner == nil {
		return nil
	}
	return e.resolveScanner()
}

// explicitVMScanners returns explicit VM scanners in deterministic integration-ID
// order so selection remains stable when multiple integrations are present.
// The current product assumption is that only one effective VM scanner should
// be used, but sorting keeps the first-selection rule predictable if more than
// one integration is configured.
func (e *enricherImpl) explicitVMScanners() []types.VirtualMachineScanner {
	e.lock.RLock()
	defer e.lock.RUnlock()

	ids := slices.Sorted(maps.Keys(e.scanners))
	resolved := make([]types.VirtualMachineScanner, 0, len(ids))
	for _, id := range ids {
		resolved = append(resolved, e.scanners[id])
	}
	return resolved
}

// UpsertVirtualMachineIntegration creates or replaces the explicit VM scanner
// for the provided image integration.
// Returning an error for unsupported types is intentional: Central should fail
// cleanly instead of silently pretending that a VM-scanner category is active.
func (e *enricherImpl) UpsertVirtualMachineIntegration(integration *storage.ImageIntegration) error {
	if integration == nil {
		return errors.New("virtual machine integration is required")
	}
	creator, ok := e.getScannerCreator(integration.GetType())
	if !ok {
		return errors.Errorf("unsupported virtual machine scanner integration type: %q", integration.GetType())
	}

	vmScanner, err := creator(integration)
	if err != nil {
		return errors.Wrapf(err, "creating virtual machine scanner for integration %q", integration.GetName())
	}

	e.lock.Lock()
	defer e.lock.Unlock()
	e.scanners[integration.GetId()] = vmScanner
	return nil
}

// RemoveVirtualMachineIntegration removes the explicit VM scanner associated
// with the provided integration ID so the enricher can fall back to the legacy
// resolver when no explicit VM scanner remains.
func (e *enricherImpl) RemoveVirtualMachineIntegration(id string) {
	e.lock.Lock()
	defer e.lock.Unlock()
	delete(e.scanners, id)
}

// getScannerCreator returns the registered VM-scanner creator for the given
// integration type.
// Creator lookup is separated so unsupported types can be reported before any
// scanner construction is attempted.
func (e *enricherImpl) getScannerCreator(scannerType string) (scanners.VirtualMachineScannerCreator, bool) {
	e.lock.RLock()
	defer e.lock.RUnlock()
	creator, ok := e.creators[scannerType]
	return creator, ok
}
