package fake

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/centralcaps"
)

const (
	precomputedReportVariants = 5
	// reportIntervalJitterPercent defines the percentage of random jitter to apply to report intervals.
	// With 0.05 (5%), a 60s interval will vary between 57s-63s, making timing more realistic.
	reportIntervalJitterPercent = 0.05

	// listenerRestartDelay is the time to wait for the listener restart to complete before populating VMs.
	// The 2s delay is chosen to be longer than typical listener restart time (~100-200ms) to provide a safety margin.
	// This is acceptable for fake workload code (but not production code).
	listenerRestartDelay = 2 * time.Second
)

type vmInfo struct {
	id       string
	vsockCID uint32
	name     string
}

type reportTemplate struct {
	packages     map[string]*v4.Package
	repositories map[string]*v4.Repository
}

// reportGenerator generates fake VM index reports using pre-built templates
type reportGenerator struct {
	templates          []reportTemplate
	currentTemplateIdx uint32
}

func newReportGenerator(numPackages, numRepos int) *reportGenerator {
	variantCount := precomputedReportVariants
	if variantCount <= 0 {
		variantCount = 1
	}

	templates := make([]reportTemplate, variantCount)
	for variant := range variantCount {
		repositories := make(map[string]*v4.Repository, numRepos)
		for i := range numRepos {
			repoID := fmt.Sprintf("repo-template-%d-%d", variant, i)
			repositories[repoID] = &v4.Repository{
				Id:   repoID,
				Name: fmt.Sprintf("repository-%d", i),
				Uri:  fmt.Sprintf("https://repo%d.example.com", i),
				Key:  fmt.Sprintf("key-%d", i),
				Cpe:  "cpe:2.3:o:redhat:enterprise_linux:9:*:fastdatapath:*:*:*:*:*", // valid CPE to also load scanner V4 matcher.
			}
		}

		packages := make(map[string]*v4.Package, numPackages)
		for i := range numPackages {
			pkgID := fmt.Sprintf("pkg-template-%d-%d", variant, i)

			packages[pkgID] = &v4.Package{
				Id:             pkgID,
				Name:           fmt.Sprintf("package%d", i),
				Version:        fmt.Sprintf("1.%d.%d", i/10, i%10),
				Kind:           "binary",
				Arch:           "amd64",
				RepositoryHint: "hash:sha256:f52ca767328e6919ec11a1da654e92743587bd3c008f0731f8c4de3af19c1830|key:199e2f91fd431d51",
				Cpe:            "cpe:2.3:o:redhat:enterprise_linux:9:*:fastdatapath:*:*:*:*:*", // valid CPE to also load scanner V4 matcher.
				PackageDb:      "sqlite:usr/share/rpm",
				Source: &v4.Package{
					Id:      pkgID,
					Name:    fmt.Sprintf("package%d", i),
					Version: fmt.Sprintf("1.%d.%d", i/10, i%10),
					Kind:    "source",
				},
				NormalizedVersion: &v4.NormalizedVersion{
					Kind: "rpm",
					V:    []int32{1, 0, 0},
				},
				Module: fmt.Sprintf("module%d", i),
			}
		}

		templates[variant] = reportTemplate{
			packages:     packages,
			repositories: repositories,
		}
	}

	return &reportGenerator{
		templates: templates,
	}
}

func (g *reportGenerator) nextTemplate() reportTemplate {
	if len(g.templates) == 0 {
		return reportTemplate{
			packages:     make(map[string]*v4.Package),
			repositories: make(map[string]*v4.Repository),
		}
	}
	idx := g.currentTemplateIdx % uint32(len(g.templates))
	g.currentTemplateIdx++
	return g.templates[idx]
}

// manageVMIndexReportsWithPopulation waits for the store to be set, populates fake VMs, then starts report generation
func (w *WorkloadManager) manageVMIndexReportsWithPopulation(ctx context.Context) {
	defer w.wg.Done()

	// Wait for all VM prerequisites (handler, store, and central reachability)
	if !w.vmPrerequisitesReady.Wait(ctx) {
		log.Error("Prerequisites not ready for VM index report WorkloadManager")
		return
	}

	// Verify capability is set after Central becomes reachable
	if !centralcaps.Has(centralsensor.VirtualMachinesSupported) {
		log.Warnf("Central is reachable but VirtualMachinesSupported capability not set. VM reports may fail.")
	}
	log.Infof("All VM prerequisites ready (handler, store, online mode), waiting for listener restart to complete before populating fake VMs")

	// RACE CONDITION FIX (only impacts local-sensor and integration tests in practice):
	// There is a race condition between the WorkloadManager and the event pipeline when traversing to Online mode.
	// If the WorkloadManager is faster than the event pipeline, it will populate the VM store before the listener is restarted.
	// When the listener restarts, it will clear the VM store. Thus, we must be sure that we wait with populating VMs until the listener restart is complete.
	if !w.waitForListenerRestart(ctx) {
		return
	}

	log.Infof("Populating fake VMs after listener restart delay")

	// Populate fake VMs now that Central is reachable and listener restart has completed
	// populateFakeVMs is synchronous, so VMs should be available immediately after it returns.
	w.populateFakeVMs()

	// A sanity check verification that at least one VM is present in the store.
	firstVsockCID := vmBaseVSOCKCID
	if w.vmStore.GetFromCID(firstVsockCID) == nil {
		log.Errorf("VM store population failed: first VM (vsockCID %d) not found after populateFakeVMs returned. Store=%p.", firstVsockCID, w.vmStore)
		return
	}
	log.Infof("VM store populated (found vsockCID %d), starting VM index report generation", firstVsockCID)

	// Initialize the report generator with the configured package/repo counts
	w.vmReportGen = newReportGenerator(
		w.workload.VMIndexReportWorkload.NumPackages,
		w.workload.VMIndexReportWorkload.NumRepositories,
	)
	w.manageVMIndexReports(ctx)
}

func (w *WorkloadManager) manageVMIndexReports(ctx context.Context) {
	// This function assumes VMs are already populated in the store
	// It starts a goroutine for each VM, each sending reports at the configured interval

	numVMs := w.workload.VMIndexReportWorkload.NumVMs
	reportInterval := w.workload.VMIndexReportWorkload.ReportInterval

	// Calculate total rate for logging
	totalRate := float64(numVMs) / reportInterval.Seconds()
	log.Infof("Starting VM index report generation: %d VMs, interval %s per VM, %d packages, %d repos",
		numVMs,
		reportInterval,
		w.workload.VMIndexReportWorkload.NumPackages,
		w.workload.VMIndexReportWorkload.NumRepositories)
	log.Infof("Total report rate: %.0f reports/second (%.2f reports/second per VM)",
		totalRate, totalRate/float64(numVMs))

	// Create VM info structures
	vms := make([]*vmInfo, numVMs)
	for i := 0; i < numVMs; i++ {
		vms[i] = &vmInfo{
			id:       fmt.Sprintf("vm-%d", i),
			vsockCID: vmBaseVSOCKCID + uint32(i),
			name:     fmt.Sprintf("fake-vm-%d", i),
		}
	}

	// Start a goroutine for each VM with jittered initial delay to spread reports across time.
	// This prevents all VMs from sending reports simultaneously (thundering herd).
	// Each VM gets an evenly distributed delay in [0, reportInterval), ensuring uniform
	// report distribution while maintaining the target aggregate rate.
	var wg sync.WaitGroup
	for i, vm := range vms {
		// Calculate initial delay to spread VMs evenly across the report interval
		initialDelay := time.Duration(float64(i) / float64(numVMs) * float64(reportInterval))
		wg.Add(1)
		go func(vm *vmInfo, delay time.Duration) {
			defer wg.Done()
			w.manageVMReportsForSingleVM(ctx, vm, reportInterval, delay)
		}(vm, initialDelay)
	}

	// Wait for all goroutines to finish (when context is cancelled)
	wg.Wait()
	log.Debugf("All VM report generation goroutines stopped")
}

func (w *WorkloadManager) sendVMIndexReport(ctx context.Context, vm *vmInfo) bool {
	// Check context before attempting to send
	if ctx.Err() != nil {
		return false
	}

	report := w.generateFakeIndexReport(vm.vsockCID, vm.id)

	if err := w.vmIndexReportHandler.Send(ctx, report); err != nil {
		// Handle shutdown gracefully: if handler is stopped or context is cancelled, exit silently
		if errors.Is(err, errox.InvariantViolation) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		// Log other errors (e.g., capability not supported, VM not found) at error level
		log.Errorf("Failed to send VM index report for %s: %v", vm.id, err)
		return true
	}
	log.Debugf("Sent VM index report for %s (vsockCID: %d)", vm.id, vm.vsockCID)
	return true
}

func (w *WorkloadManager) generateFakeIndexReport(vsockCID uint32, vmID string) *v1.IndexReport {
	template := w.vmReportGen.nextTemplate()

	return &v1.IndexReport{
		VsockCid: fmt.Sprintf("%d", vsockCID),
		IndexV4: &v4.IndexReport{
			HashId:  fmt.Sprintf("hash-%s", vmID),
			State:   "IndexFinished",
			Success: true,
			Contents: &v4.Contents{
				Packages:     template.packages,
				Repositories: template.repositories,
				Environments: map[string]*v4.Environment_List{
					"1": {
						Environments: []*v4.Environment{
							{
								PackageDb:     "sqlite:usr/share/rpm",
								IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
								RepositoryIds: []string{"cpe:/o:redhat:enterprise_linux:9::fastdatapath", "cpe:/a:redhat:openshift:4.16::el9"},
							},
						},
					},
				},
			},
		},
	}
}

// jitteredInterval returns a duration with random jitter applied.
// The result is in the range [interval * (1 - jitterPercent), interval * (1 + jitterPercent)].
// For example, with interval=60s and jitterPercent=0.05, returns a value between 57s and 63s.
func jitteredInterval(interval time.Duration, jitterPercent float64) time.Duration {
	// Calculate jitter range: interval * jitterPercent
	jitterRange := float64(interval) * jitterPercent
	// Random value in [-jitterRange, +jitterRange]
	jitter := (rand.Float64()*2 - 1) * jitterRange
	// Return interval with jitter applied
	return time.Duration(float64(interval) + jitter)
}

func (w *WorkloadManager) manageVMReportsForSingleVM(ctx context.Context, vm *vmInfo, interval, initialDelay time.Duration) {
	// Apply initial delay to spread reports across time (startup jitter)
	if initialDelay > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(initialDelay):
		}
	}

	// Send first report after the initial delay
	if !w.sendVMIndexReport(ctx, vm) {
		return
	}

	// Continue sending reports with jittered intervals to simulate realistic timing variance
	for {
		// Calculate next report time with jitter
		nextInterval := jitteredInterval(interval, reportIntervalJitterPercent)

		select {
		case <-ctx.Done():
			return
		case <-time.After(nextInterval):
			if !w.sendVMIndexReport(ctx, vm) {
				return
			}
		}
	}
}

// waitForListenerRestart waits for the listener restart delay to ensure the store is ready.
// Returns true if the wait completed, false if the context was cancelled.
func (w *WorkloadManager) waitForListenerRestart(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(listenerRestartDelay):
		return true
	}
}

// repopulateVMsOnOnlineTransition repopulates the VM store after an offline→online transition.
// This is necessary because the listener restart clears all stores (including vmStore).
// The method waits for the listener restart delay before repopulating to avoid a race condition.
func (w *WorkloadManager) repopulateVMsOnOnlineTransition(ctx context.Context) {
	if !w.waitForListenerRestart(ctx) {
		log.Debug("Context cancelled while waiting for listener restart during VM repopulation")
		return
	}

	countBefore := w.vmStore.Size()
	log.Infof("Repopulating fake VMs after offline→online transition (store size before: %d)", countBefore)

	w.populateFakeVMs()

	countAfter := w.vmStore.Size()
	log.Infof("VM store repopulation complete (store size: %d before → %d after, expected: %d)",
		countBefore, countAfter, w.workload.VMIndexReportWorkload.NumVMs)
}
