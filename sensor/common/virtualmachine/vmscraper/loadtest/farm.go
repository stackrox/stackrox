// Package loadtest provides a synthetic VM "farm" used to stress-test the
// VSOCK pull-mode VMScraper without any real KubeVirt infrastructure.
//
// It reuses real production code on both ends of the protocol: each FarmVM
// wraps a real roxagent vsockserver.Handler + ReportCache, and the harness
// drives a real vmscraper.VMScraper against them. Only the outer edges
// (KubeVirt, Central, actual filesystem scanning) are faked.
//
// See docs/superpowers/specs/2026-07-03-vsock-pull-stress-test-design.md.
package loadtest

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsockserver"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/fixtures/vmindexreport"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
)

var log = logging.LoggerForModule()

// Namespace used for all synthetic VMs.
const Namespace = "loadtest"

// reportGeneratorSeed keeps package selection reproducible across runs.
const reportGeneratorSeed = int64(42)

// FarmVM is one simulated VM: a real vsockserver.Handler + ReportCache
// serving a generated fixture report, standing in for roxagent inside a VM.
type FarmVM struct {
	Namespace string
	Name      string
	VSOCKCID  uint32
	Handler   *vsockserver.Handler

	cache  *vsockserver.ReportCache
	report *v4.IndexReport
	facts  map[string]string

	// lastScrape is a UnixNano timestamp of the last successful dial to this
	// VM, used for backlog tracking (see Farm.BacklogCount). Initialized to
	// creation time so a freshly added VM isn't immediately "overdue".
	lastScrape atomic.Int64
}

// key returns the "namespace/name" identity used to look up a FarmVM.
func (vm *FarmVM) key() string { return vm.Namespace + "/" + vm.Name }

// bump republishes the same report content, incrementing the generation
// counter (SetReport always increments, regardless of content).
func (vm *FarmVM) bump() {
	vm.cache.SetReport(vm.report, vm.facts)
}

// markScraped records that this VM was just successfully contacted.
//
// intentional simplification: called on every successful dial, regardless
// of whether the poll turns out to be "changed" or "unchanged" -- there is
// no failure injection yet, so a successful dial in this harness is
// equivalent to a successful end-to-end scrape. Upgrade path: move this to
// after a confirmed successful read once failure injection exists.
func (vm *FarmVM) markScraped() {
	vm.lastScrape.Store(time.Now().UnixNano())
}

// Farm manages a fleet of synthetic VMs and implements vmscraper.RunningVMStore.
type Farm struct {
	rescanInterval time.Duration
	alwaysChanged  bool
	gen            *vmindexreport.Generator

	mu        sync.RWMutex
	vms       map[string]*FarmVM
	nextIndex int
}

// NewFarm creates numVMs synthetic VMs, each seeded with a report containing
// numPackages packages (reusing the real ~450 KiB/524-package fixture shape
// used elsewhere for VM load testing).
//
// rescanInterval controls how often a randomly chosen VM bumps its report
// generation (0 disables rescanning, i.e. every poll after the first
// observes "unchanged"). It is ignored when alwaysChanged is true.
//
// alwaysChanged, if true, bumps a VM's generation on every single dial to it
// (see FarmDialer), so VMScraper always takes the full-report path instead
// of "unchanged". Without this, VMScraper's per-VM generation cache means
// only the very first poll of each VM is ever a real report -- everything
// after that is a cheap unchanged-ack unless the VM happens to be the one
// picked by the rescan loop, which at fleet sizes beyond a handful of VMs
// means the expensive report-parsing path is barely exercised at all.
func NewFarm(numVMs, numPackages int, rescanInterval time.Duration, alwaysChanged bool) *Farm {
	gen := vmindexreport.NewGeneratorWithSeed(numPackages, reportGeneratorSeed)
	f := &Farm{rescanInterval: rescanInterval, alwaysChanged: alwaysChanged, gen: gen, vms: make(map[string]*FarmVM, numVMs)}
	for range numVMs {
		vm := newFarmVM(f.nextIndex, gen)
		f.vms[vm.key()] = vm
		f.nextIndex++
	}
	log.Infof("loadtest farm: created %d synthetic VMs, %d packages/report, always_changed=%t", numVMs, numPackages, alwaysChanged)
	return f
}

// AddVMs grows the farm by n additional synthetic VMs, without disturbing
// any existing VM's state. Used by ramp mode to step the fleet size up
// mid-run instead of tearing down and recreating the whole farm.
func (f *Farm) AddVMs(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for range n {
		vm := newFarmVM(f.nextIndex, f.gen)
		f.vms[vm.key()] = vm
		f.nextIndex++
	}
}

// Count returns the current number of VMs in the farm.
func (f *Farm) Count() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.vms)
}

// BacklogCount returns the number of VMs whose time since last successful
// scrape exceeds pollInterval -- the ramp-mode saturation signal: a Sensor
// that's keeping up with the fleet sees this stay near zero, while one that
// can't sees it grow without bound (see design spec, "Backlog gauge").
func (f *Farm) BacklogCount(pollInterval time.Duration) int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	now := time.Now()
	count := 0
	for _, vm := range f.vms {
		if now.Sub(time.Unix(0, vm.lastScrape.Load())) > pollInterval {
			count++
		}
	}
	return count
}

func newFarmVM(index int, gen *vmindexreport.Generator) *FarmVM {
	report := gen.GenerateV4IndexReport()
	facts := map[string]string{"detected_os": "RHEL", "os_version": "9.4"}
	cache := &vsockserver.ReportCache{}
	cache.SetReport(report, facts)
	vm := &FarmVM{
		Namespace: Namespace,
		Name:      fmt.Sprintf("vm-%d", index),
		VSOCKCID:  uint32(1000 + index),
		Handler:   vsockserver.NewHandler(cache, "loadtest-agent"),
		cache:     cache,
		report:    report,
		facts:     facts,
	}
	vm.lastScrape.Store(time.Now().UnixNano())
	return vm
}

// GetByName returns the FarmVM for namespace/name, or nil if not found. If the
// farm was created with alwaysChanged, the VM's generation is bumped first
// so the caller's next dial always observes a "changed" report.
func (f *Farm) GetByName(namespace, name string) *FarmVM {
	f.mu.RLock()
	defer f.mu.RUnlock()
	vm := f.vms[namespace+"/"+name]
	if vm != nil && f.alwaysChanged {
		vm.bump()
	}
	return vm
}

// Get implements vmscraper.RunningVMStore.
func (f *Farm) Get(id virtualmachine.VMID) *virtualmachine.Info {
	f.mu.RLock()
	defer f.mu.RUnlock()
	vm := f.vms[string(id)]
	if vm == nil {
		return nil
	}
	cid := vm.VSOCKCID
	return &virtualmachine.Info{
		ID:        virtualmachine.VMID(vm.key()),
		Name:      vm.Name,
		Namespace: vm.Namespace,
		VSOCKCID:  &cid,
		Running:   true,
	}
}

// ListRunning implements vmscraper.RunningVMStore.
func (f *Farm) ListRunning() []*virtualmachine.Info {
	f.mu.RLock()
	defer f.mu.RUnlock()
	infos := make([]*virtualmachine.Info, 0, len(f.vms))
	for _, vm := range f.vms {
		cid := vm.VSOCKCID
		infos = append(infos, &virtualmachine.Info{
			ID:        virtualmachine.VMID(vm.key()),
			Name:      vm.Name,
			Namespace: vm.Namespace,
			VSOCKCID:  &cid,
			Running:   true,
		})
	}
	return infos
}

// Start begins the background rescan loop. It returns immediately; the loop
// stops when ctx is cancelled. It is a no-op when alwaysChanged is set,
// since every dial already bumps its VM's generation (see GetByName).
func (f *Farm) Start(ctx context.Context) {
	if f.alwaysChanged || f.rescanInterval <= 0 {
		return
	}
	go f.rescanLoop(ctx)
}

func (f *Farm) rescanLoop(ctx context.Context) {
	ticker := time.NewTicker(f.rescanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f.rescanOneRandomVM()
		}
	}
}

// rescanOneRandomVM bumps a single VM's report generation per tick, staggering
// generation changes across cycles rather than bumping the whole fleet at
// once.
//
// intentional simplification: exactly one VM per tick regardless of fleet
// size, so churn rate does not scale with VM count. Upgrade path: bump a
// configurable fraction of the fleet per tick if a more realistic
// fleet-wide change rate is needed.
func (f *Farm) rescanOneRandomVM() {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if len(f.vms) == 0 {
		return
	}
	target := rand.Intn(len(f.vms))
	i := 0
	for _, vm := range f.vms {
		if i == target {
			vm.bump()
			return
		}
		i++
	}
}
