package fake

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"

	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/errox"
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

	// Wait for handler to be set using Signal
	// Check if already set first to handle race condition where it's set before we start waiting
	if w.vmIndexReportHandler == nil {
		log.Debugf("Waiting for VM index report handler to be set")
		select {
		case <-ctx.Done():
			return
		case <-w.vmHandlerReady.Done():
			// Handler was set, verify it's not nil
			if w.vmIndexReportHandler == nil {
				log.Errorf("Received handler ready signal but handler is still nil")
				return
			}
		case <-time.After(readinessCheckInterval * time.Duration(maxVMStoreWaitAttempts)):
			log.Errorf("Timeout waiting for VM index report handler after %v", readinessCheckInterval*time.Duration(maxVMStoreWaitAttempts))
			return
		}
	}
	log.Debugf("VM index report handler set")

	// Wait for store to be set using Signal
	// Check if already set first to handle race condition where it's set before we start waiting
	if w.vmStore == nil {
		log.Debugf("Waiting for VM store to be set")
		select {
		case <-ctx.Done():
			return
		case <-w.vmStoreReady.Done():
			// Store was set, verify it's not nil
			if w.vmStore == nil {
				log.Errorf("Received store ready signal but store is still nil")
				return
			}
		case <-time.After(readinessCheckInterval * time.Duration(maxVMStoreWaitAttempts)):
			log.Errorf("Timeout waiting for VM store after %v", readinessCheckInterval*time.Duration(maxVMStoreWaitAttempts))
			return
		}
	}
	log.Debugf("VM store is set (store=%p), waiting for Central to be reachable", w.vmStore)

	// Wait for Central to be reachable (WorkloadManager receives SensorComponentEventCentralReachable via Notify())
	// This is more reliable than polling for capabilities, as it properly handles connection retries.
	log.Debugf("Waiting for Central to be reachable...")
	select {
	case <-ctx.Done():
		return
	case <-w.centralReachable.Done():
		// Central is reachable, verify capability is also set
		if !centralcaps.Has(centralsensor.VirtualMachinesSupported) {
			log.Warnf("Central is reachable but VirtualMachinesSupported capability not set. VM reports may fail.")
		} else {
			log.Debugf("Central is reachable and VM capability is available")
		}
	case <-time.After(readinessCheckInterval * time.Duration(maxCapabilityWaitAttempts)):
		log.Errorf("Timeout waiting for Central to be reachable after %v. VM reports will not be sent. This may indicate that the connection to Central failed or SensorComponentEventCentralReachable was not received.", readinessCheckInterval*time.Duration(maxCapabilityWaitAttempts))
		return
	}
	log.Infof("Central is reachable, populating fake VMs")

	// Populate fake VMs now that Central is reachable and we're ready to send reports
	// populateFakeVMs is synchronous and includes verification, so VMs should be available immediately after it returns
	w.populateFakeVMs()

	// Quick verification that at least the first VM is present
	// This is a sanity check - populateFakeVMs should have already verified the VMs
	firstVsockCID := vmBaseVSOCKCID
	if w.vmStore.GetFromCID(firstVsockCID) == nil {
		log.Errorf("VM store population failed: first VM (vsockCID %d) not found after populateFakeVMs returned. Store=%p.", firstVsockCID, w.vmStore)
		return
	}
	log.Infof("VM store populated (found vsockCID %d), starting VM index report generation", firstVsockCID)

	// Now start the actual report generation loop
	w.manageVMIndexReports(ctx)
}

func (w *WorkloadManager) manageVMIndexReports(ctx context.Context) {
	// This function assumes VMs are already populated in the store
	// It starts a goroutine for each VM, each sending reports at the configured interval

	vmPool := newVMPool(w.workload.VMIndexReportWorkload.NumVMs)
	reportInterval := w.workload.VMIndexReportWorkload.ReportInterval
	numVMs := w.workload.VMIndexReportWorkload.NumVMs

	// Calculate total rate for logging
	totalRate := float64(numVMs) / reportInterval.Seconds()
	log.Infof("Starting VM index report generation: %d VMs, interval %s per VM, %d packages, %d repos",
		numVMs,
		reportInterval,
		w.workload.VMIndexReportWorkload.NumPackages,
		w.workload.VMIndexReportWorkload.NumRepositories)
	log.Infof("Total report rate: %.0f reports/second (%.2f reports/second per VM)",
		totalRate, totalRate/float64(numVMs))

	// Start a goroutine for each VM
	var wg sync.WaitGroup
	for i := 0; i < numVMs; i++ {
		vm := vmPool.vms[i]
		wg.Add(1)
		go func(vm *vmInfo) {
			defer wg.Done()
			w.manageVMReportsForSingleVM(ctx, vm, reportInterval)
		}(vm)
	}

	// Wait for all goroutines to finish (when context is cancelled)
	wg.Wait()
	log.Debugf("All VM report generation goroutines stopped")
}

func (w *WorkloadManager) manageVMReportsForSingleVM(ctx context.Context, vm *vmInfo, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		// Check context again before attempting to send (in case it was cancelled during tick)
		if ctx.Err() != nil {
			return
		}

		report := generateFakeIndexReport(vm,
			w.workload.VMIndexReportWorkload.NumPackages,
			w.workload.VMIndexReportWorkload.NumRepositories)

		if err := w.vmIndexReportHandler.Send(ctx, report); err != nil {
			// Handle shutdown gracefully: if handler is stopped or context is cancelled, exit silently
			if errors.Is(err, errox.InvariantViolation) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			// Log other errors (e.g., capability not supported, VM not found) at error level
			log.Errorf("Failed to send VM index report for %s: %v", vm.id, err)
		} else {
			log.Debugf("Sent VM index report for %s (vsockCID: %d)", vm.id, vm.vsockCID)
		}
	}
}
