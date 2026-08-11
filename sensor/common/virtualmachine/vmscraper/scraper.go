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
	tickInterval          time.Duration
	reconcileEvery        time.Duration
	perVMTimeout          time.Duration
	mandatoryRefreshAfter time.Duration
	concurrency           int
	warnMaxBytes          int
	stopper               concurrency.Stopper
	started               atomic.Bool
	now                   func() time.Time

	mu            sync.Mutex
	vmState       map[string]*vmState
	lastReconcile time.Time
	inFlight      set.StringSet
}

type vmState struct {
	lastGeneration  uint32
	lastEpoch       uint32
	lastForwardedAt time.Time
	nextAttemptAt   time.Time
	backoff         time.Duration
	vmID            virtualmachine.VMID
}

var _ common.SensorComponent = (*VMScraper)(nil)

// New creates a VMScraper with production defaults.
func New(store RunningVMStore, sender IndexReportSender, dialer VMDialer, client ProtocolClient) *VMScraper {
	interval := clampPollInterval(env.VirtualMachinesScraperPollInterval.DurationSetting())
	return &VMScraper{
		store:                 store,
		sender:                sender,
		dialer:                dialer,
		client:                client,
		interval:              interval,
		tickInterval:          initialBackoff,
		reconcileEvery:        reconcilePeriod(interval),
		perVMTimeout:          env.VirtualMachinesScraperPerVMTimeout.DurationSetting(),
		mandatoryRefreshAfter: env.VirtualMachinesScraperMandatoryRefreshInterval.DurationSetting(),
		concurrency:           env.VirtualMachinesScraperConcurrency.IntegerSetting(),
		// Warn once a report is halfway to the hard response-size ceiling,
		// so operators get advance notice before reports start actually
		// being rejected at that limit.
		warnMaxBytes: env.VirtualMachinesPullMaxResponseSizeKB.IntegerSetting() * 1024 / 2,
		vmState:      make(map[string]*vmState),
		inFlight:     set.NewStringSet(),
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

// handleNACK clears the cached generation and applies the shared backoff so
// the next tick resends a full report without tight-looping on persistent NACKs.
//
// Race with commitVMState after Send is accepted (same class as today's
// lastGeneration race): a fast NACK may be overwritten by a late success commit.
func (s *VMScraper) handleNACK(resourceID string) {
	key := s.findKeyByVMID(vmIDFromResourceID(resourceID))
	if key == "" {
		log.Debugf("VMScraper: could not resolve VM for NACKed resource_id=%q; nothing to reset", resourceID)
		return
	}

	var backoff time.Duration
	ok := concurrency.WithLock1(&s.mu, func() bool {
		state, exists := s.vmState[key]
		if !exists {
			return false
		}
		state.lastGeneration = 0
		state.backoff = nextBackoff(state.backoff, s.interval)
		state.nextAttemptAt = s.now().Add(state.backoff)
		backoff = state.backoff
		return true
	})
	if ok {
		log.Infof("VMScraper: NACK for %q; cleared generation, next attempt in %s", key, backoff)
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

	// Reconcile + scrape immediately so VMs don't wait a full tick on start.
	s.pollOnce(ctx)

	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx, false)
		}
	}
}

// pollOnce forces a reconcile and scrapes every due slot. Tests call this
// directly; production uses tick() on the short ticker.
func (s *VMScraper) pollOnce(ctx context.Context) {
	s.tick(ctx, true)
}

func (s *VMScraper) tick(ctx context.Context, forceReconcile bool) {
	cycleStart := s.now()
	reconcile := forceReconcile
	if !reconcile {
		concurrency.WithLock(&s.mu, func() {
			reconcile = s.lastReconcile.IsZero() || s.now().Sub(s.lastReconcile) >= s.reconcileEvery
		})
	}
	if reconcile {
		s.reconcile()
	}

	due := s.dueKeys()
	log.Debugf("VMScraper: tick: %d due VMs %v (concurrency=%d, reconcile=%v)", len(due), due, s.concurrency, reconcile)
	if reconcile {
		metrics.PullCyclesTotal.Inc()
		concurrency.WithLock(&s.mu, func() {
			metrics.PullVMsInCycle.Set(float64(len(s.vmState)))
		})
	}

	var successCount atomic.Int32
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(s.concurrency)

	for _, key := range due {
		g.Go(func() error {
			if s.scrapeKey(gCtx, key) {
				successCount.Add(1)
			}
			return nil
		})
	}
	_ = g.Wait()

	elapsed := time.Since(cycleStart)
	if reconcile {
		log.Debugf("VMScraper: tick done: %d/%d due VMs succeeded in %s", successCount.Load(), len(due), elapsed.Truncate(time.Millisecond))
		metrics.PullCycleDurationSeconds.Observe(elapsed.Seconds())
	}
}

func (s *VMScraper) reconcile() {
	vms := s.store.ListRunning()
	liveKeys := set.NewStringSet()
	concurrency.WithLock(&s.mu, func() {
		now := s.now()
		for _, vm := range vms {
			key := vmKey(vm)
			liveKeys.Add(key)
			st, ok := s.vmState[key]
			if !ok {
				st = &vmState{nextAttemptAt: now, vmID: vm.ID}
				s.vmState[key] = st
			}
			st.vmID = vm.ID
		}
		for key := range s.vmState {
			if !liveKeys.Contains(key) {
				delete(s.vmState, key)
			}
		}
		s.lastReconcile = now
	})
}

func (s *VMScraper) dueKeys() []string {
	return concurrency.WithLock1(&s.mu, func() []string {
		now := s.now()
		var due []string
		for key, st := range s.vmState {
			if s.inFlight.Contains(key) {
				continue
			}
			if st.nextAttemptAt.After(now) {
				continue
			}
			due = append(due, key)
		}
		return due
	})
}

func (s *VMScraper) scrapeKey(ctx context.Context, key string) bool {
	claimed := concurrency.WithLock1(&s.mu, func() bool {
		if s.inFlight.Contains(key) {
			return false
		}
		s.inFlight.Add(key)
		return true
	})
	if !claimed {
		return false
	}
	defer concurrency.WithLock(&s.mu, func() { s.inFlight.Remove(key) })

	vmID := concurrency.WithLock1(&s.mu, func() virtualmachine.VMID {
		if st, ok := s.vmState[key]; ok {
			return st.vmID
		}
		return ""
	})
	vm := s.store.Get(vmID)
	if vm == nil || !vm.Running {
		concurrency.WithLock(&s.mu, func() {
			delete(s.vmState, key)
		})
		return false
	}
	return s.scrapeVM(ctx, vm)
}

func (s *VMScraper) scrapeVM(ctx context.Context, vm *virtualmachine.Info) bool {
	key := vmKey(vm)
	snap := s.snapshotVMState(key)
	reason, backoff := s.scrapeScheduleHint(key)
	log.Infof("VMScraper: scraping %q (reason=%s backoff=%s)", key, reason, backoff)

	vmCtx, cancel := context.WithTimeout(ctx, s.perVMTimeout)
	defer cancel()

	totalStart := s.now()
	port := getVsockPort()

	// The mandatory refresh is a deterministic, time-based decision Sensor
	// can make before dialing at all, so request the full report on this
	// round trip instead of asking "anything newer?" and dialing again.
	ifNewerThan, knownEpoch := snap.lastGeneration, snap.lastEpoch
	mandatoryRefreshDue := s.now().Sub(snap.lastForwardedAt) > s.mandatoryRefreshAfter
	if mandatoryRefreshDue {
		ifNewerThan, knownEpoch = 0, 0
	}

	log.Debugf("VMScraper: dialing roxagent on %q with TLS", key)
	result, outcome := s.dialAndGetReport(vmCtx, vm, key, port, ifNewerThan, knownEpoch)
	if outcome != scrapeOK {
		next := s.scheduleAfterAttempt(key, outcome)
		kind := "retryable"
		if outcome == scrapeNonRetryable {
			kind = "non-retryable"
		}
		log.Infof("VMScraper: scrape %q failed %s next=%s", key, kind, next)
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
			next := s.scheduleAfterAttempt(key, scrapeOK)
			log.Infof("VMScraper: scrape %q ok outcome=unchanged next=%s", key, next)
			log.Debugf("VMScraper: unchanged report from roxagent on %q (generation=%d)", key, snap.lastGeneration)
			metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusUnchanged).Inc()
			return true
		}
		log.Infof("VMScraper: roxagent on %q restarted (epoch changed from %d to %d, generation coincidentally matched cached value %d) — forcing full report",
			key, snap.lastEpoch, epoch, snap.lastGeneration)
		result, outcome = s.dialAndGetReport(vmCtx, vm, key, port, 0, 0)
		if outcome != scrapeOK {
			next := s.scheduleAfterAttempt(key, outcome)
			kind := "retryable"
			if outcome == scrapeNonRetryable {
				kind = "non-retryable"
			}
			log.Infof("VMScraper: scrape %q failed %s next=%s", key, kind, next)
			return false
		}
	}

	viable, warning := reportcheck.IsViable(result.IndexReport, s.warnMaxBytes)
	if warning != "" {
		log.Warnf("VM report from %q: %s", key, warning)
	}
	if !viable {
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusInvalidReport).Inc()
		next := s.scheduleAfterAttempt(key, scrapeNonRetryable)
		log.Infof("VMScraper: scrape %q failed non-retryable next=%s", key, next)
		return false
	}

	reportSize := proto.Size(result.IndexReport)
	metrics.PullReportBytes.Observe(float64(reportSize))
	metrics.PullReportPackages.Observe(float64(len(result.IndexReport.GetContents().GetPackages())))
	recordVMDiscoveredData(result.Meta.GetFacts())

	if err := s.sender.Send(vmCtx, vm, result.IndexReport); err != nil {
		log.Errorf("VMScraper: sending %q report to Central failed: %v", key, err)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusSendError).Inc()
		next := s.scheduleAfterAttempt(key, scrapeNonRetryable)
		log.Infof("VMScraper: scrape %q failed non-retryable next=%s", key, next)
		return false
	}

	newGen := result.Meta.GetReportGeneration()
	s.commitVMState(key, newGen, result.Meta.GetEpoch())
	next := s.scheduleAfterAttempt(key, scrapeOK)

	log.Infof("VMScraper: scrape %q ok outcome=forwarded next=%s", key, next)
	log.Debugf("VMScraper: successfully pulled report for %q: generation=%d, packages=%d, size=%d bytes, total=%s",
		key, newGen, len(result.IndexReport.GetContents().GetPackages()), reportSize,
		time.Since(totalStart).Truncate(time.Millisecond))
	metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusSuccess).Inc()
	metrics.PullTotalDurationSeconds.Observe(time.Since(totalStart).Seconds())
	return true
}

// scrapeScheduleHint reports whether this scrape is a normal poll or a backoff
// retry, based on the current per-VM backoff (NACK and retryable failures share
// that field).
func (s *VMScraper) scrapeScheduleHint(key string) (string, time.Duration) {
	return concurrency.WithLock2(&s.mu, func() (string, time.Duration) {
		st, ok := s.vmState[key]
		if !ok || st.backoff <= 0 {
			return "poll", 0
		}
		return "retry", st.backoff
	})
}

type scrapeOutcome int

const (
	scrapeOK scrapeOutcome = iota
	scrapeRetryable
	scrapeNonRetryable
)

// scheduleAfterAttempt updates nextAttemptAt/backoff for key and returns the
// delay until the next attempt (0 if the slot was dropped).
func (s *VMScraper) scheduleAfterAttempt(key string, outcome scrapeOutcome) time.Duration {
	return concurrency.WithLock1(&s.mu, func() time.Duration {
		st, ok := s.vmState[key]
		if !ok {
			return 0
		}
		now := s.now()
		switch outcome {
		case scrapeOK:
			st.backoff = 0
			st.nextAttemptAt = now.Add(s.interval)
			return s.interval
		case scrapeRetryable:
			st.backoff = nextBackoff(st.backoff, s.interval)
			st.nextAttemptAt = now.Add(st.backoff)
			return st.backoff
		case scrapeNonRetryable:
			st.backoff = 0
			st.nextAttemptAt = now.Add(s.interval)
			return s.interval
		default:
			return 0
		}
	})
}

// dialAndGetReport dials the VM and issues a single GetReport request,
// recording dial/read latency metrics and classifying failures.
func (s *VMScraper) dialAndGetReport(ctx context.Context, vm *virtualmachine.Info, key string, port, ifNewerThan, knownEpoch uint32) (*vsockclient.GetReportResult, scrapeOutcome) {
	dialStart := s.now()
	stream, err := s.dialer.Dial(ctx, vm.Namespace, vm.Name, port, true)
	metrics.PullDialDurationSeconds.Observe(time.Since(dialStart).Seconds())
	if err != nil {
		if ctx.Err() != nil {
			log.Warnf("VMScraper: dialing roxagent on %q timed out: %v", key, err)
			metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusTimeout).Inc()
			if errors.Is(ctx.Err(), context.Canceled) {
				// Parent cancel (Sensor stop): do not keep retrying on the short tick.
				return nil, scrapeNonRetryable
			}
			return nil, scrapeRetryable
		}
		log.Warnf("VMScraper: dialing roxagent on %q failed: %v", key, err)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusDialError).Inc()
		return nil, scrapeRetryable
	}
	defer func() { _ = stream.Close() }()

	readStart := s.now()
	result, err := s.client.GetReport(ctx, stream, ifNewerThan, knownEpoch)
	metrics.PullReadDurationSeconds.Observe(time.Since(readStart).Seconds())
	if err != nil {
		return nil, s.handleGetReportError(ctx, key, err)
	}
	return result, scrapeOK
}

func (s *VMScraper) handleGetReportError(ctx context.Context, key string, err error) scrapeOutcome {
	// Timed-out or cancelled reads surface here after Dial's deadline
	// and/or GetReport's close-on-cancel. Prefer ctx.Err() so those land
	// as timeout (matching the dial path) rather than as a protocol/read error.
	if ctx.Err() != nil {
		log.Warnf("VMScraper: reading report from %q timed out: %v", key, err)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusTimeout).Inc()
		if errors.Is(ctx.Err(), context.Canceled) {
			return scrapeNonRetryable
		}
		return scrapeRetryable
	}
	switch {
	case errors.Is(err, vsockclient.ErrNotReady):
		log.Debugf("VMScraper: roxagent on %q has not yet generated a report", key)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusNotReady).Inc()
		return scrapeRetryable
	case errors.Is(err, vsockclient.ErrUnknownMethod):
		log.Warnf("VMScraper: roxagent on %q does not support the GetReport method", key)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusUnknownMethod).Inc()
		return scrapeNonRetryable
	case errors.Is(err, vsockclient.ErrInternal):
		log.Warnf("VMScraper: roxagent on %q reported an internal error: %v", key, err)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusReadError).Inc()
		return scrapeRetryable
	case errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF):
		log.Debugf("VMScraper: roxagent on %q connection closed (agent may be down or restarting): %v", key, err)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusReadError).Inc()
		return scrapeRetryable
	case errors.Is(err, vsockclient.ErrBusy):
		log.Infof("VMScraper: roxagent on %q is busy with another request: %v", key, err)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusBusy).Inc()
		return scrapeRetryable
	default:
		log.Warnf("VMScraper: protocol error for %q (possible version mismatch): %v", key, err)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusReadError).Inc()
		if isRetryable(err) {
			return scrapeRetryable
		}
		return scrapeNonRetryable
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
			st = &vmState{nextAttemptAt: s.now()}
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
// then the NACK will overwrite lastGeneration / backoff / nextAttemptAt, and then this code
// will set generation (and a later scheduleAfterAttempt(scrapeOK) will reset backoff schedule).
// This race is accepted for v1 (same class as the historical lastGeneration-only race).
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

// recordVMDiscoveredData increments VMDiscoveredData from ResponseMeta.facts
// keys written by roxagent (detected_os, activation_status, dnf_metadata_status).
func recordVMDiscoveredData(facts map[string]string) {
	metrics.VMDiscoveredData.WithLabelValues(
		facts["detected_os"],
		facts["activation_status"],
		facts["dnf_metadata_status"],
	).Inc()
}
