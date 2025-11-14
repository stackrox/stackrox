package fake

import (
	"context"
	"fmt"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"

	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
)

func generateFakeIndexReport(vm *vmInfo, numPackages, numRepos int) *v1.IndexReport {
	packages := make(map[string]*v4.Package)
	repositories := make(map[string]*v4.Repository)

	// Generate repositories first
	for i := 0; i < numRepos; i++ {
		repoID := fmt.Sprintf("repo-%s-%d", vm.id, i)
		repositories[repoID] = &v4.Repository{
			Id:   repoID,
			Name: fmt.Sprintf("repository-%d", i),
			Uri:  fmt.Sprintf("https://repo%d.example.com", i),
			Key:  fmt.Sprintf("key-%d", i),
		}
	}

	// Generate packages
	for i := 0; i < numPackages; i++ {
		pkgID := fmt.Sprintf("pkg-%s-%d", vm.id, i)
		repoHint := ""
		if numRepos > 0 {
			repoHint = fmt.Sprintf("repo-%s-%d", vm.id, i%numRepos)
		}

		packages[pkgID] = &v4.Package{
			Id:             pkgID,
			Name:           fmt.Sprintf("package%d", i),
			Version:        fmt.Sprintf("1.%d.%d", i/10, i%10),
			Kind:           "binary",
			Arch:           "amd64",
			RepositoryHint: repoHint,
		}
	}

	return &v1.IndexReport{
		VsockCid: fmt.Sprintf("%d", vm.vsockCID),
		IndexV4: &v4.IndexReport{
			HashId:  fmt.Sprintf("hash-%s", vm.id),
			State:   "IndexFinished",
			Success: true,
			Contents: &v4.Contents{
				Packages:     packages,
				Repositories: repositories,
			},
		},
	}
}

// manageVMIndexReportsWithPopulation waits for the store to be set, populates fake VMs, then starts report generation
func (w *WorkloadManager) manageVMIndexReportsWithPopulation(ctx context.Context) {
	if w.workload.VMIndexReportWorkload.NumVMs == 0 ||
		w.workload.VMIndexReportWorkload.ReportInterval == 0 {
		return
	}

	// Wait for handler to be set
	for w.vmIndexReportHandler == nil {
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
			log.Infof("Waiting for VM index report handler to be set, retrying in 100ms")
		}
	}
	log.Infof("VM index report handler set")

	// Wait for store to be set
	for w.vmStore == nil {
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
			log.Infof("Waiting for VM store to be set, retrying in 100ms")
		}
	}
	log.Infof("VM store is set (store=%p), populating fake VMs", w.vmStore)

	// Populate fake VMs now that store is set
	w.populateFakeVMs()

	// Verify VMs are populated before starting report generation
	firstVsockCID := uint32(1000)
	maxWaitAttempts := 50 // 5 seconds max wait
	attempts := 0
	for w.vmStore.GetFromCID(firstVsockCID) == nil {
		attempts++
		if attempts > maxWaitAttempts {
			log.Errorf("Timeout waiting for VM store to be populated after %d attempts. Store=%p.", maxWaitAttempts, w.vmStore)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
			log.Infof("Waiting for VM store to be populated (checking vsockCID %d, attempt %d/%d), retrying in 100ms", firstVsockCID, attempts, maxWaitAttempts)
		}
	}
	log.Infof("VM store populated (found vsockCID %d), waiting for Central VM capability", firstVsockCID)

	// Wait for Central to advertise VirtualMachinesSupported capability
	// This is necessary because capabilities are set when CentralHello is received,
	// which happens asynchronously during the gRPC stream handshake in initialSync().
	// Capabilities are set in centralcaps.Set() when CentralHello is processed.
	maxCapabilityWaitAttempts := 100 // 10 seconds max wait (initialSync can take a few seconds)
	capabilityAttempts := 0
	for !centralcaps.Has(centralsensor.VirtualMachinesSupported) {
		capabilityAttempts++
		if capabilityAttempts > maxCapabilityWaitAttempts {
			log.Errorf("Timeout waiting for Central VM capability after %d attempts (10s). VM reports will not be sent. This may indicate that CentralHello was not received or processed correctly.", maxCapabilityWaitAttempts)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
			if capabilityAttempts%10 == 0 {
				// Log every second to avoid spam
				log.Infof("Waiting for Central VM capability (attempt %d/%d), retrying in 100ms", capabilityAttempts, maxCapabilityWaitAttempts)
			}
		}
	}
	log.Infof("Central VM capability available, starting VM index report generation")

	// Now start the actual report generation loop
	w.manageVMIndexReports(ctx)
}

func (w *WorkloadManager) manageVMIndexReports(ctx context.Context) {
	// This function assumes VMs are already populated in the store
	// It just starts generating reports

	vmPool := newVMPool(w.workload.VMIndexReportWorkload.NumVMs)
	ticker := time.NewTicker(w.workload.VMIndexReportWorkload.ReportInterval)
	defer ticker.Stop()

	log.Infof("Starting VM index report generation: %d VMs, interval %s, %d packages, %d repos",
		w.workload.VMIndexReportWorkload.NumVMs,
		w.workload.VMIndexReportWorkload.ReportInterval,
		w.workload.VMIndexReportWorkload.NumPackages,
		w.workload.VMIndexReportWorkload.NumRepositories)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		vm := vmPool.getRoundRobin()
		report := generateFakeIndexReport(vm,
			w.workload.VMIndexReportWorkload.NumPackages,
			w.workload.VMIndexReportWorkload.NumRepositories)

		if err := w.vmIndexReportHandler.Send(ctx, report); err != nil {
			log.Errorf("Failed to send VM index report for %s: %v", vm.id, err)
		} else {
			log.Debugf("Sent VM index report for %s (vsockCID: %d)", vm.id, vm.vsockCID)
		}
	}
}
