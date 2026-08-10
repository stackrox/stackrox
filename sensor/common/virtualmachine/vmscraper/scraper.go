package vmscraper

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
	"github.com/stackrox/rox/sensor/common/virtualmachine/reportcheck"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

var (
	log                  = logging.LoggerForModule()
	errStartMoreThanOnce = errors.New("unable to start the VM scraper more than once")
)

// minPollInterval clamps ROX_VIRTUAL_MACHINES_SCRAPER_POLL_INTERVAL to avoid
// accidental high-churn vsock polling.
const minPollInterval = time.Minute

func getVsockPort() uint32 {
	return uint32(env.VirtualMachinesVsockPort.IntegerSetting())
}

func clampPollInterval(interval time.Duration) time.Duration {
	if interval < minPollInterval {
		log.Warnf("ROX_VIRTUAL_MACHINES_SCRAPER_POLL_INTERVAL=%v is below the minimum of %v; using %v",
			interval, minPollInterval, minPollInterval)
		return minPollInterval
	}
	return interval
}

// RunningVMStore provides lookups of running VMs.
type RunningVMStore interface {
	ListRunning() []*virtualmachine.Info
	Get(id virtualmachine.VMID) *virtualmachine.Info
}

// VMDialer connects to a VM's VSOCK port.
type VMDialer interface {
	Dial(ctx context.Context, namespace, name string, port uint32, useTLS bool) (io.ReadWriteCloser, error)
}

// IndexReportSender sends index reports toward Central.
type IndexReportSender interface {
	Send(ctx context.Context, vm *virtualmachine.Info, report *v4.IndexReport) error
}

// ProtocolClient performs the request/response protocol over a stream.
type ProtocolClient interface {
	GetReport(ctx context.Context, stream io.ReadWriteCloser, ifNewerThan uint32, knownEpoch uint32) (*vsockclient.GetReportResult, error)
}

// VMScraper polls running VMs and pulls their scan reports via VSOCK.
type VMScraper struct {
	store                 RunningVMStore
	sender                IndexReportSender
	dialer                VMDialer
	client                ProtocolClient
	interval              time.Duration
	perVMTimeout          time.Duration
	mandatoryRefreshAfter time.Duration
	concurrency           int
	warnMaxBytes          int
	stopper               concurrency.Stopper
	started               atomic.Bool
	now                   func() time.Time

	mu      sync.Mutex
	vmState map[string]*vmState
}

type vmState struct {
	lastGeneration  uint32
	lastEpoch       uint32
	lastForwardedAt time.Time
}

var _ common.SensorComponent = (*VMScraper)(nil)

// New creates a VMScraper with production defaults.
func New(store RunningVMStore, sender IndexReportSender, dialer VMDialer, client ProtocolClient) *VMScraper {
	return &VMScraper{
		store:                 store,
		sender:                sender,
		dialer:                dialer,
		client:                client,
		interval:              clampPollInterval(env.VirtualMachinesScraperPollInterval.DurationSetting()),
		perVMTimeout:          env.VirtualMachinesScraperPerVMTimeout.DurationSetting(),
		mandatoryRefreshAfter: env.VirtualMachinesScraperMandatoryRefreshInterval.DurationSetting(),
		concurrency:           env.VirtualMachinesScraperConcurrency.IntegerSetting(),
		// Warn once a report is halfway to the hard response-size ceiling,
		// so operators get advance notice before reports start actually
		// being rejected at that limit.
		warnMaxBytes: env.VirtualMachinesPullMaxResponseSizeKB.IntegerSetting() * 1024 / 2,
		vmState:      make(map[string]*vmState),
		now:          time.Now,
	}
}

func (s *VMScraper) Name() string { return "virtualmachine.vmscraper" }

func (s *VMScraper) Start() error {
	if s.started.Swap(true) {
		return errStartMoreThanOnce
	}
	s.stopper = concurrency.NewStopper()
	go s.run()
	return nil
}

func (s *VMScraper) Stop() {
	s.stopper.Client().Stop()
}

func (s *VMScraper) Capabilities() []centralsensor.SensorCapability { return nil }
func (s *VMScraper) Notify(_ common.SensorComponentEvent)           {}

// Accepts reports whether this component wants to see SensorACK messages
// for VM index reports.
func (s *VMScraper) Accepts(msg *central.MsgToSensor) bool {
	sensorAck := msg.GetSensorAck()
	return sensorAck != nil && sensorAck.GetMessageType() == central.SensorACK_VM_INDEX_REPORT
}

// ProcessMessage handles SensorACK/NACK for pull-mode VM index reports.
func (s *VMScraper) ProcessMessage(_ context.Context, msg *central.MsgToSensor) error {
	sensorAck := msg.GetSensorAck()
	if sensorAck == nil || sensorAck.GetMessageType() != central.SensorACK_VM_INDEX_REPORT {
		return nil
	}

	// Record all action types for debuggability; label cardinality risk is accepted.
	metrics.IndexReportAcksReceived.WithLabelValues(sensorAck.GetAction().String()).Inc()

	switch sensorAck.GetAction() {
	case central.SensorACK_ACK:
		log.Debugf("VMScraper: received acknowledgement for resource_id=%q", sensorAck.GetResourceId())
	case central.SensorACK_NACK:
		s.handleNACK(sensorAck.GetResourceId())
	default:
		log.Debugf("VMScraper: received unknown SensorACK action %v for resource_id=%q", sensorAck.GetAction(), sensorAck.GetResourceId())
	}
	return nil
}

// handleNACK clears the cached generation so the next poll resends a full report.
func (s *VMScraper) handleNACK(resourceID string) {
	key := s.findKeyByVMID(vmIDFromResourceID(resourceID))
	if key == "" {
		log.Debugf("VMScraper: could not resolve VM for NACKed resource_id=%q; nothing to reset", resourceID)
		return
	}

	reset := concurrency.WithLock1(&s.mu, func() bool {
		state, ok := s.vmState[key]
		if !ok {
			return false
		}
		state.lastGeneration = 0
		return true
	})
	if reset {
		log.Debugf("VMScraper: reset cached generation for %q after NACK for resource_id=%q", key, resourceID)
	}
}

// findKeyByVMID returns the vmState key ("namespace/name") for the currently
// running VM with the given ID, or "" if none matches.
func (s *VMScraper) findKeyByVMID(vmID string) string {
	if vmID == "" {
		return ""
	}
	vm := s.store.Get(virtualmachine.VMID(vmID))
	if vm == nil || !vm.Running {
		return ""
	}
	return vmKey(vm)
}

// vmIDFromResourceID extracts the VM ID from a composite ACK resource ID
// (format "vmID:vsockCID"; see central/sensor/service/common.VMIndexACKResourceID).
func vmIDFromResourceID(resourceID string) string {
	vmID, _, _ := strings.Cut(resourceID, ":")
	return vmID
}

func (s *VMScraper) ResponsesC() <-chan *message.ExpiringMessage { return nil }

func (s *VMScraper) run() {
	defer s.stopper.Flow().ReportStopped()
	ctx := concurrency.AsContext(s.stopper.LowLevel().GetStopRequestSignal())

	// Poll immediately on start so VMs don't wait a full interval before first scrape.
	s.pollOnce(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pollOnce(ctx)
		}
	}
}

func (s *VMScraper) pollOnce(ctx context.Context) {
	cycleStart := s.now()
	vms := s.store.ListRunning()
	log.Infof("VMScraper: about to poll %d running VMs (concurrency=%d)", len(vms), s.concurrency)
	metrics.PullCyclesTotal.Inc()
	metrics.PullVMsInCycle.Set(float64(len(vms)))

	var successCount atomic.Int32

	liveKeys := set.NewStringSet()
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(s.concurrency)

	for _, vm := range vms {
		liveKeys.Add(vmKey(vm))
		g.Go(func() error {
			if s.scrapeVM(gCtx, vm) {
				successCount.Add(1)
			}
			return nil
		})
	}
	_ = g.Wait()

	s.pruneStaleVMState(liveKeys)
	elapsed := time.Since(cycleStart)
	log.Infof("VMScraper: cycle done: %d/%d VMs scraped successfully in %s", successCount.Load(), len(vms), elapsed.Truncate(time.Millisecond))
	metrics.PullCycleDurationSeconds.Observe(elapsed.Seconds())
}

func (s *VMScraper) scrapeVM(ctx context.Context, vm *virtualmachine.Info) bool {
	key := vmKey(vm)
	snap := s.snapshotVMState(key)

	vmCtx, cancel := context.WithTimeout(ctx, s.perVMTimeout)
	defer cancel()

	totalStart := s.now()
	port := getVsockPort()

	// The mandatory refresh is a deterministic, time-based decision Sensor
	// can make before dialing at all, so request the full report on this
	// first (and only) round trip instead of asking "anything newer?" and
	// re-dialing once told no.
	ifNewerThan, knownEpoch := snap.lastGeneration, snap.lastEpoch
	mandatoryRefreshDue := s.now().Sub(snap.lastForwardedAt) > s.mandatoryRefreshAfter
	if mandatoryRefreshDue {
		ifNewerThan, knownEpoch = 0, 0
	}

	log.Debugf("VMScraper: dialing roxagent on %q with TLS", key)
	// knownEpoch lets a current roxagent make the complete "changed or not"
	// decision server-side and serve the full report in this same round trip
	// on a restart-coincidence false match, instead of relying solely on the
	// client-side fallback below.
	result, ok := s.dialAndGetReport(vmCtx, vm, key, port, ifNewerThan, knownEpoch)
	if !ok {
		return false
	}

	if result.Unchanged {
		// Backward-compat fallback: against a current roxagent that honors
		// knownEpoch, an epoch mismatch is already resolved in the response
		// above (Unchanged would be false), so this branch is effectively
		// dead code. It only matters against an agent that predates the
		// knownEpoch request field and ignores it, where roxagent's
		// generation-only comparison can produce a false "unchanged" right
		// after a restart (report_generation resets to 1 and coincidentally
		// re-matches Sensor's cached value).
		//
		// mandatoryRefreshDue is deliberately not re-checked here: when
		// true, the call above already requested ifNewerThan=0, so a
		// current roxagent can only report Unchanged for a reason other
		// than the mandatory refresh — i.e. an epoch mismatch below.
		epoch := result.Meta.GetEpoch()
		epochMismatch := epoch != 0 && epoch != snap.lastEpoch
		if !epochMismatch {
			log.Debugf("VMScraper: unchanged report from roxagent on %q (generation=%d)", key, snap.lastGeneration)
			metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusUnchanged).Inc()
			return true
		}
		log.Infof("VMScraper: roxagent on %q restarted (epoch changed from %d to %d, generation coincidentally matched cached value %d) — forcing full report",
			key, snap.lastEpoch, epoch, snap.lastGeneration)
		result, ok = s.dialAndGetReport(vmCtx, vm, key, port, 0, 0)
		if !ok {
			return false
		}
	}

	viable, warning := reportcheck.IsViable(result.IndexReport, s.warnMaxBytes)
	if warning != "" {
		log.Warnf("VM report from %q: %s", key, warning)
	}
	if !viable {
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusInvalidReport).Inc()
		return false
	}

	reportSize := proto.Size(result.IndexReport)
	metrics.PullReportBytes.Observe(float64(reportSize))
	metrics.PullReportPackages.Observe(float64(len(result.IndexReport.GetContents().GetPackages())))
	recordVMDiscoveredData(result.Meta.GetFacts())

	if err := s.sender.Send(vmCtx, vm, result.IndexReport); err != nil {
		log.Errorf("VMScraper: sending %q report to Central failed: %v", key, err)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusSendError).Inc()
		return false
	}

	newGen := result.Meta.GetReportGeneration()
	s.commitVMState(key, newGen, result.Meta.GetEpoch())

	log.Debugf("VMScraper: successfully pulled report for %q: generation=%d, packages=%d, size=%d bytes, total=%s",
		key, newGen, len(result.IndexReport.GetContents().GetPackages()), reportSize,
		time.Since(totalStart).Truncate(time.Millisecond))
	metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusSuccess).Inc()
	metrics.PullTotalDurationSeconds.Observe(time.Since(totalStart).Seconds())
	return true
}

// dialAndGetReport dials the VM and issues a single GetReport request,
// recording dial/read latency metrics and classifying dial failures
// (timeout vs. other) consistently regardless of which call site invokes
// it. scrapeVM calls this twice on the epoch-mismatch fallback path (see
// the Unchanged branch above), so keeping the timing and error-handling
// logic in one place ensures both calls are measured the same way.
//
// Because each call observes PullDialDurationSeconds and
// PullReadDurationSeconds, that fallback path produces two histogram
// samples for one logical VM scrape. PullRequestsTotal is still
// incremented once (on the final outcome). Dashboard authors should not
// equate sum(rate(..._count)) of those histograms with PullRequestsTotal.
// The second dial (and thus the double observation) goes away once
// ROX-35756 replaces the generation+epoch pair with a single per-scan
// token, which removes the restart-coincidence fallback entirely.
func (s *VMScraper) dialAndGetReport(ctx context.Context, vm *virtualmachine.Info, key string, port, ifNewerThan, knownEpoch uint32) (*vsockclient.GetReportResult, bool) {
	dialStart := s.now()
	stream, err := s.dialer.Dial(ctx, vm.Namespace, vm.Name, port, true)
	metrics.PullDialDurationSeconds.Observe(time.Since(dialStart).Seconds())
	if err != nil {
		if ctx.Err() != nil {
			log.Warnf("VMScraper: dialing roxagent on %q timed out: %v", key, err)
			metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusTimeout).Inc()
		} else {
			log.Warnf("VMScraper: dialing roxagent on %q failed: %v", key, err)
			metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusDialError).Inc()
		}
		return nil, false
	}
	defer func() { _ = stream.Close() }()

	readStart := s.now()
	result, err := s.client.GetReport(ctx, stream, ifNewerThan, knownEpoch)
	metrics.PullReadDurationSeconds.Observe(time.Since(readStart).Seconds())
	if err != nil {
		s.handleGetReportError(ctx, key, err)
		return nil, false
	}
	return result, true
}

func (s *VMScraper) handleGetReportError(ctx context.Context, key string, err error) {
	// Timed-out or cancelled reads surface here after Dial's deadline
	// and/or GetReport's close-on-cancel. Prefer ctx.Err() so those land
	// as timeout (matching the dial path) rather than as a protocol/read error.
	if ctx.Err() != nil {
		log.Warnf("VMScraper: reading report from %q timed out: %v", key, err)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusTimeout).Inc()
		return
	}
	switch {
	case errors.Is(err, vsockclient.ErrNotReady):
		log.Debugf("VMScraper: roxagent on %q has not yet generated a report", key)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusNotReady).Inc()
	case errors.Is(err, vsockclient.ErrUnknownMethod):
		log.Warnf("VMScraper: roxagent on %q does not support the GetReport method", key)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusUnknownMethod).Inc()
	case errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF):
		log.Debugf("VMScraper: roxagent on %q connection closed (agent may be down or restarting): %v", key, err)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusReadError).Inc()
	case errors.Is(err, vsockclient.ErrBusy):
		log.Infof("VMScraper: roxagent on %q is busy with another request: %v", key, err)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusBusy).Inc()
	default:
		log.Warnf("VMScraper: protocol error for %q (possible version mismatch): %v", key, err)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusReadError).Inc()
	}
}

// vmKey returns the identifier used for vmState lookups.
func vmKey(vm *virtualmachine.Info) string {
	return vm.Namespace + "/" + vm.Name
}

type vmStateSnapshot struct {
	lastGeneration  uint32
	lastEpoch       uint32
	lastForwardedAt time.Time
}

func (s *VMScraper) snapshotVMState(key string) vmStateSnapshot {
	return concurrency.WithLock1(&s.mu, func() vmStateSnapshot {
		st, ok := s.vmState[key]
		if !ok {
			st = &vmState{}
			s.vmState[key] = st
		}
		return vmStateSnapshot{
			lastGeneration:  st.lastGeneration,
			lastEpoch:       st.lastEpoch,
			lastForwardedAt: st.lastForwardedAt,
		}
	})
}

// commitVMState records the generation/epoch from a just-sent report as key's cached scrape state.
//
// Race: If a NACK arrives from Central faster than this function completes, for example, due to:
//   - a scheduling delay on the caller's goroutine between Send returning and this function
//     acquiring `s.mu` (a GC pause, CPU throttling, or GOMAXPROCS contention),
//   - contention on `s.mu` itself, from other concurrent scrapes or NACKs delaying this
//     function's lock acquisition,
//
// then the NACK will overwrite lastGeneration to 0, and then this code will set it back to
// `newGen`, which cancels the NACK.
// However, this race is really unlikely to happen in practice, thus we accept the risk and not protect against it.
func (s *VMScraper) commitVMState(key string, newGen, newEpoch uint32) {
	concurrency.WithLock(&s.mu, func() {
		state, ok := s.vmState[key]
		if !ok {
			return
		}
		state.lastGeneration = newGen
		state.lastEpoch = newEpoch
		state.lastForwardedAt = s.now()
	})
}

func (s *VMScraper) pruneStaleVMState(liveKeys set.StringSet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key := range s.vmState {
		if !liveKeys.Contains(key) {
			delete(s.vmState, key)
		}
	}
}

// recordVMDiscoveredData increments VMDiscoveredData from ResponseMeta.facts
// keys written by roxagent (detected_os, activation_status, dnf_metadata_status).
func recordVMDiscoveredData(facts map[string]string) {
	metrics.VMDiscoveredData.WithLabelValues(
		facts["detected_os"],
		facts["activation_status"],
		facts["dnf_metadata_status"],
	).Inc()
}
