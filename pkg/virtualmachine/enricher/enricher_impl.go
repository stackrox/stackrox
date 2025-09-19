package enricher

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scannerv4/client"
)

var (
	log         = logging.LoggerForModule()
	scanTimeout = env.ScanTimeout.DurationSetting()
	vmDigest    name.Digest
)

const vmMockDigest = "vm-registry/repository@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f"

type enricherImpl struct {
	scannerClient client.Scanner
}

func New(scannerClient client.Scanner) VirtualMachineEnricher {
	return &enricherImpl{
		scannerClient: scannerClient,
	}
}

func (e *enricherImpl) EnrichVirtualMachineWithVulnerabilities(vm *storage.VirtualMachine, indexReport *v4.IndexReport) error {
	// Clear any pre-existing notes
	vm.Notes = vm.Notes[:0]

	if e.scannerClient == nil {
		vm.Notes = append(vm.Notes, storage.VirtualMachine_MISSING_SCAN_DATA)
		return errors.New("Scanner V4 client not available for VM enrichment")
	}

	if indexReport == nil {
		vm.Notes = append(vm.Notes, storage.VirtualMachine_MISSING_SCAN_DATA)
		return errors.New("index report is required for VM scanning")
	}

	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()

	vr, err := e.scannerClient.GetVulnerabilities(ctx, vmDigest, indexReport.GetContents())
	if err != nil {
		vm.Notes = append(vm.Notes, storage.VirtualMachine_MISSING_SCAN_DATA)
		return errors.Wrap(err, "failed to get vulnerability report for VM")
	}

	vm.Scan = toVirtualMachineScan(vr)
	log.Debugf("Enriched VM %s with %d components", vm.GetName(), len(vm.GetScan().GetComponents()))
	return nil
}

func init() {
	vmDigest, err := name.NewDigest(vmMockDigest)
	if err != nil {
		panic(fmt.Sprintf(err, "failed to parse mock digest %q", vmDigest))
	}
}
