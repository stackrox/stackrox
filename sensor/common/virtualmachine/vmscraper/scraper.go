package vmscraper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
	"github.com/stackrox/rox/sensor/common/virtualmachine/reportcheck"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

var (
	log                       = logging.LoggerForModule()
	errStartMoreThanOnce      = errors.New("unable to start the VM scraper more than once")
	errCapabilityNotSupported = errors.New("Central does not have virtual machine capability")
	errCentralNotReachable    = errors.New("Central is not reachable")
)

const (
	// minPollInterval clamps ROX_VIRTUAL_MACHINES_SCRAPER_POLL_INTERVAL to avoid
	// accidental high-churn vsock polling.
	minPollInterval = time.Minute

	sendNotImplementedLogLimiter = "vm-scraper-send-not-implemented"
)

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
	AddOrUpdate(vm *virtualmachine.Info) *virtualmachine.Info
}

// VMDialer connects to a VM's VSOCK port.
type VMDialer interface {
	Dial(ctx context.Context, namespace, name string, port uint32, useTLS bool) (io.ReadWriteCloser, error)
}

// ProtocolClient performs the request/response protocol over a stream.
type ProtocolClient interface {
	GetReport(ctx context.Context, stream io.ReadWriteCloser, lastKnownToken string) (*vsockclient.GetReportResult, error)
	SyncRepoCPEMapping(ctx context.Context, stream io.ReadWriteCloser, mapping []byte) (updated bool, meta *pb.ResponseMeta, err error)
}

// Repo2CPEFetcher supplies Sensor's cached repo-to-CPE mapping so maybeSyncRepoCPEMapping
// can tell whether a VM's agent needs a pushed update.
type Repo2CPEFetcher interface {
	FetchRepo2CPE(ctx context.Context) (mapping []byte, hash string, ok bool)
}

type clusterIDGetter interface {
	GetNoWait() string
}

// closeCoder is satisfied by transport errors carrying a structured close
// code. Declared locally so VMScraper doesn't need to import a concrete
// dialer's error type to recognize one.
type closeCoder interface {
	error
	CloseCode() (code int, reason string)
}

// VMScraper polls running VMs and pulls their scan reports via VSOCK.
type VMScraper struct {
	store                 RunningVMStore
	dialer                VMDialer
	client                ProtocolClient
	repo2CPEFetcher       Repo2CPEFetcher
	clusterID             clusterIDGetter
	toCentral             chan *message.ExpiringMessage
	centralReady          concurrency.Signal
	interval              time.Duration
	tickInterval          time.Duration
	initialBackoff        time.Duration
	reconcileEvery        time.Duration
	perVMTimeout          time.Duration
	mandatoryRefreshAfter time.Duration
	concurrency           int
	spreadFraction        float64
	warnMaxBytes          int
	stopper               concurrency.Stopper
	started               atomic.Bool
	// loggedSkip is set after the first skip log for a missing-capability stretch
	// so a 10s ticker does not repeat it until the capability returns.
	loggedSkip atomic.Bool
	now        func() time.Time
	// randFloat64 returns a unit sample in [0, 1] for schedule offsets; tests inject a fixed source.
	randFloat64          func() float64
	lastSpreadWarnNumVMs int

	mu              sync.Mutex
	vmState         map[string]*vmState
	lastReconcile   time.Time
	inFlight        set.StringSet
	lastForwardTime time.Time // Sensor-level; for forward interarrival metric
}

type vmState struct {
	lastToken        string
	lastForwardedAt  time.Time
	nextAttemptAt    time.Time
	backoff          time.Duration
	vmID             virtualmachine.VMID
	lastAgentVersion string
}

var _ common.SensorComponent = (*VMScraper)(nil)

// New creates a VMScraper with production defaults. A nil repo2CPEFetcher leaves
// maybeSyncRepoCPEMapping a no-op, so pull still works if the repo-to-CPE cache was not built.
func New(store RunningVMStore, dialer VMDialer, client ProtocolClient, repo2CPEFetcher Repo2CPEFetcher, clusterID clusterIDGetter) *VMScraper {
	interval := clampPollInterval(env.VirtualMachinesScraperPollInterval.DurationSetting())
	return &VMScraper{
		store:                 store,
		dialer:                dialer,
		client:                client,
		repo2CPEFetcher:       repo2CPEFetcher,
		clusterID:             clusterID,
		toCentral:             make(chan *message.ExpiringMessage, env.VirtualMachinesIndexReportsBufferSize.IntegerSetting()),
		centralReady:          concurrency.NewSignal(),
		interval:              interval,
		tickInterval:          env.VirtualMachinesScraperTickInterval.DurationSetting(),
		initialBackoff:        env.VirtualMachinesScraperInitialBackoff.DurationSetting(),
		reconcileEvery:        reconcilePeriod(interval),
		perVMTimeout:          env.VirtualMachinesScraperPerVMTimeout.DurationSetting(),
		mandatoryRefreshAfter: env.VirtualMachinesScraperMandatoryRefreshInterval.DurationSetting(),
		concurrency:           env.VirtualMachinesScraperConcurrency.IntegerSetting(),
		spreadFraction:        env.VirtualMachinesScraperSteadySpreadFraction.FloatSetting(),
		// Warn once a report is halfway to the hard response-size ceiling,
		// so operators get advance notice before reports start actually
		// being rejected at that limit.
		warnMaxBytes: env.VirtualMachinesPullMaxResponseSizeKB.IntegerSetting() * 1024 / 2,
		vmState:      make(map[string]*vmState),
		inFlight:     set.NewStringSet(),
		now:          time.Now,
		randFloat64:  rand.Float64,
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

func (s *VMScraper) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.SensorACKSupport}
}

func (s *VMScraper) Notify(e common.SensorComponentEvent) {
	switch e {
	case common.SensorComponentEventCentralReachable:
		s.centralReady.Signal()
	case common.SensorComponentEventOfflineMode:
		s.centralReady.Reset()
	}
}

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

// handleNACK clears the cached token and applies the shared backoff so
// the next tick resends a full report without tight-looping on persistent NACKs.
//
// Race with commitVMState after forward is accepted: a fast NACK may be
// overwritten by a late success commit.
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
		state.lastToken = ""
		state.backoff = nextBackoff(state.backoff, s.interval, s.initialBackoff)
		state.nextAttemptAt = s.now().Add(state.backoff)
		backoff = state.backoff
		return true
	})
	if ok {
		log.Infof("VMScraper: NACK for %q; cleared token, next attempt in %s", key, backoff)
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
	return vm.Key()
}

// vmIDFromResourceID extracts the VM ID from a composite ACK resource ID
// (format "vmID:vsockCID"; see central/sensor/service/common.VMIndexACKResourceID).
func vmIDFromResourceID(resourceID string) string {
	vmID, _, _ := strings.Cut(resourceID, ":")
	return vmID
}

func (s *VMScraper) ResponsesC() <-chan *message.ExpiringMessage { return s.toCentral }

func (s *VMScraper) run() {
	defer s.stopper.Flow().ReportStopped()
	ctx := concurrency.AsContext(s.stopper.LowLevel().GetStopRequestSignal())

	// Block until stop: there is nothing to poll, but the component stays registered.
	if s.dialer == nil || s.client == nil {
		<-ctx.Done()
		return
	}

	// Reconcile + scrape immediately so VMs don't wait a full tick on start.
	s.tick(ctx, true)

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

func (s *VMScraper) tick(ctx context.Context, forceReconcile bool) {
	if !centralcaps.Has(centralsensor.VirtualMachinesSupported) {
		if s.loggedSkip.CompareAndSwap(false, true) {
			log.Infof("VMScraper: skipping pulling index reports from VMs; Central does not advertise VirtualMachinesSupported")
		}
		return
	}
	s.loggedSkip.Store(false)

	tickStart := s.now()
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
	nDue := len(due)
	nTracked := concurrency.WithLock1(&s.mu, func() int {
		n := len(s.vmState)
		metrics.PullTrackedVMs.Set(float64(n))
		return n
	})
	metrics.PullDueVMs.Set(float64(nDue))
	capN := min(s.concurrency, startBudget(nTracked, s.tickInterval, newVMIndexReportWindow(s.interval)))
	if capN < nDue {
		due = due[:capN]
	}
	started := len(due)
	// Idle ticks stay silent so a 10s scheduler step does not look like an empty cluster.
	if started > 0 {
		log.Debugf("VMScraper: tick: %d due of %d tracked, starting %d %v (concurrency=%d, reconcile=%v)",
			nDue, nTracked, started, due, s.concurrency, reconcile)
		metrics.PullStartsPerTick.Observe(float64(started))
	}
	metrics.PullTicksTotal.Inc()

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

	elapsed := s.now().Sub(tickStart)
	if started > 0 {
		log.Debugf("VMScraper: tick done: %d/%d started of %d tracked succeeded in %s",
			successCount.Load(), started, nTracked, elapsed.Truncate(time.Millisecond))
	}
	metrics.PullTickDurationSeconds.Observe(elapsed.Seconds())
}

// reconcile syncs vmState with currently running VMs: drop gone ones, and
// schedule first attempts for new ones across the initial scan window (new
// guests are often still booting; Sensor restart rehydrates many at once).
func (s *VMScraper) reconcile() {
	vms := s.store.ListRunning()
	liveKeys := set.NewStringSet()
	var numVMs int
	concurrency.WithLock(&s.mu, func() {
		now := s.now()
		newVMWindow := newVMIndexReportWindow(s.interval)
		for _, vm := range vms {
			key := vm.Key()
			liveKeys.Add(key)
			st, ok := s.vmState[key]
			if !ok {
				st = &vmState{nextAttemptAt: now.Add(randOffset(newVMWindow, s.randFloat64())), vmID: vm.ID}
				s.vmState[key] = st
			} else if st.vmID != vm.ID {
				// namespace/name can outlive a KubeVirt recreate; do not inherit scrape state.
				st = &vmState{nextAttemptAt: now, vmID: vm.ID}
				s.vmState[key] = st
			}
		}
		for key := range s.vmState {
			if !liveKeys.Contains(key) {
				delete(s.vmState, key)
			}
		}
		numVMs = len(s.vmState)
		s.lastReconcile = now
	})
	s.warnIfSpreadSaturated(numVMs)
}

// warnIfSpreadSaturated logs when running VMs exceed what this Sensor's
// cadence spread can serialize at one scrape per tick.
func (s *VMScraper) warnIfSpreadSaturated(numVMs int) {
	if s.tickInterval <= 0 || s.lastSpreadWarnNumVMs == numVMs {
		return
	}
	s.lastSpreadWarnNumVMs = numVMs
	capacity := int(steadySpreadWidth(s.interval, s.spreadFraction) / s.tickInterval)
	if capacity <= 0 || numVMs <= capacity {
		return
	}
	log.Warnf("VMScraper: %d running VMs exceed what ROX_VIRTUAL_MACHINES_SCRAPER_POLL_INTERVAL=%s can space out on this Sensor (about %d); index reports may be forwarded in bursts.",
		numVMs, s.interval, capacity)
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
			if st, ok := s.vmState[key]; ok && st.vmID == vmID {
				delete(s.vmState, key)
			}
		})
		return false
	}
	return s.scrapeVM(ctx, vm)
}

func (s *VMScraper) scrapeVM(ctx context.Context, vm *virtualmachine.Info) bool {
	key := vm.Key()
	snap := s.snapshotVMState(key)
	reason := "poll"
	if snap.backoff > 0 {
		reason = "retry"
	}
	log.Infof("VMScraper: scraping %q (reason=%s backoff=%s)", key, reason, snap.backoff)

	totalStart := s.now()
	port := getVsockPort()

	// The mandatory refresh is a deterministic, time-based decision Sensor
	// can make before dialing at all, so request the full report on this
	// round trip instead of asking "anything newer?" and dialing again.
	lastKnownToken := snap.lastToken
	mandatoryRefreshDue := s.now().Sub(snap.lastForwardedAt) > s.mandatoryRefreshAfter
	if mandatoryRefreshDue {
		lastKnownToken = ""
	}

	log.Debugf("VMScraper: dialing roxagent on %q with TLS", key)
	result, outcome := s.dialAndGetReport(ctx, vm, key, port, lastKnownToken)
	if outcome != scrapeOK {
		next := s.scheduleAfterAttempt(key, vm.ID, outcome)
		kind := "retryable"
		if outcome == scrapeNonRetryable {
			kind = "non-retryable"
		}
		log.Infof("VMScraper: scrape %q failed %s next=%s", key, kind, next)
		return false
	}

	if result.Unchanged {
		s.forwardAgentFactsIfChanged(ctx, vm, result.Meta.GetFacts())
		next := s.scheduleAfterAttempt(key, vm.ID, scrapeOK)
		log.Infof("VMScraper: scrape %q ok outcome=unchanged next=%s", key, next)
		log.Debugf("VMScraper: unchanged report from roxagent on %q (token=%s)", key, snap.lastToken)
		metrics.PullGetReportTotal.WithLabelValues(metrics.PullGetReportUnchanged).Inc()
		return true
	}

	viable, warning := reportcheck.IsViable(result.IndexReport, s.warnMaxBytes)
	if warning != "" {
		log.Warnf("VM report from %q: %s", key, warning)
	}
	if !viable {
		metrics.PullScrapeTotal.WithLabelValues(metrics.PullScrapeInvalidReport).Inc()
		next := s.scheduleAfterAttempt(key, vm.ID, scrapeNonRetryable)
		log.Infof("VMScraper: scrape %q failed non-retryable next=%s", key, next)
		return false
	}

	reportSize := proto.Size(result.IndexReport)
	metrics.PullReportBytes.Observe(float64(reportSize))
	metrics.PullReportPackages.Observe(float64(len(result.IndexReport.GetContents().GetPackages())))
	logAndRecordDiscoveredFacts(key, result.Meta.GetFacts())
	s.persistAgentFacts(vm, result.Meta.GetFacts())

	// Mapping sync has its own deadline, so forward uses the scrape parent.
	if err := s.forwardReport(ctx, vm, result.IndexReport); err != nil {
		metrics.PullScrapeTotal.WithLabelValues(metrics.PullScrapeSendError).Inc()
		outcome := scrapeRetryable
		if errors.Is(err, errox.NotImplemented) {
			logging.GetRateLimitedLogger().WarnL(sendNotImplementedLogLimiter,
				"VMScraper: Central cannot consume VM index reports: %v", err)
			outcome = scrapeNonRetryable
		} else {
			log.Errorf("VMScraper: sending %q report to Central failed: %v", key, err)
		}
		// Send failures are typically a transient Central connection issue, so
		// retry on the short backoff rather than waiting a full poll interval.
		next := s.scheduleAfterAttempt(key, vm.ID, outcome)
		kind := "retryable"
		if outcome == scrapeNonRetryable {
			kind = "non-retryable"
		}
		log.Infof("VMScraper: scrape %q failed %s next=%s", key, kind, next)
		return false
	}
	s.tryEnqueueVMUpdate(vm)

	newToken := result.Meta.GetReportToken()
	s.commitVMState(key, vm.ID, newToken, result.Meta.GetAgentVersion())
	s.observeForwardInterarrival()
	next := s.scheduleAfterAttempt(key, vm.ID, scrapeOK)

	log.Infof("VMScraper: scrape %q ok outcome=forwarded next=%s", key, next)
	totalElapsed := s.now().Sub(totalStart)
	log.Debugf("VMScraper: successfully pulled report for %q: token=%s, packages=%d, size=%d bytes, total=%s",
		key, newToken, len(result.IndexReport.GetContents().GetPackages()), reportSize,
		totalElapsed.Truncate(time.Millisecond))
	metrics.PullScrapeTotal.WithLabelValues(metrics.PullScrapeSuccess).Inc()
	metrics.PullTotalDurationSeconds.Observe(totalElapsed.Seconds())
	return true
}

// observeForwardInterarrival records the Sensor-level gap between successful
// forwards. The first forward after start does not observe a sample.
func (s *VMScraper) observeForwardInterarrival() {
	concurrency.WithLock(&s.mu, func() {
		now := s.now()
		if !s.lastForwardTime.IsZero() {
			metrics.PullForwardInterarrivalSeconds.Observe(now.Sub(s.lastForwardTime).Seconds())
		}
		s.lastForwardTime = now
	})
}

type scrapeOutcome int

const (
	scrapeOK scrapeOutcome = iota
	scrapeRetryable
	scrapeNonRetryable
)

// scheduleAfterAttempt updates nextAttemptAt/backoff for key and returns the
// delay until the next attempt (0 if the slot was dropped or belongs to another VM).
// Retries use short exponential backoff; success / permanent failure return to
// poll cadence with a random offset in [0, steadyWidth].
func (s *VMScraper) scheduleAfterAttempt(key string, vmID virtualmachine.VMID, outcome scrapeOutcome) time.Duration {
	return concurrency.WithLock1(&s.mu, func() time.Duration {
		st, ok := s.vmState[key]
		if !ok || st.vmID != vmID {
			return 0
		}
		now := s.now()
		switch outcome {
		case scrapeRetryable:
			st.backoff = nextBackoff(st.backoff, s.interval, s.initialBackoff)
			st.nextAttemptAt = now.Add(st.backoff)
			return st.backoff
		case scrapeOK, scrapeNonRetryable:
			st.backoff = 0
			offset := randOffset(steadySpreadWidth(s.interval, s.spreadFraction), s.randFloat64())
			metrics.PullScheduleOffsetSeconds.Observe(offset.Seconds())
			delay := s.interval + offset
			st.nextAttemptAt = now.Add(delay)
			return delay
		default:
			return 0
		}
	})
}

// dialAndGetReport uses a separate perVMTimeout for GetReport and for the
// mapping push so a slow report cannot abort the push.
func (s *VMScraper) dialAndGetReport(ctx context.Context, vm *virtualmachine.Info, key string, port uint32, lastKnownToken string) (*vsockclient.GetReportResult, scrapeOutcome) {
	reportCtx, cancel := context.WithTimeout(ctx, s.perVMTimeout)
	defer cancel()

	dialStart := s.now()
	stream, err := s.dialer.Dial(reportCtx, vm.Namespace, vm.Name, port, true)
	metrics.PullDialDurationSeconds.Observe(s.now().Sub(dialStart).Seconds())
	if err != nil {
		if reportCtx.Err() != nil {
			log.Warnf("VMScraper: dialing roxagent on %q timed out: %v", key, err)
			metrics.PullTransportTotal.WithLabelValues(metrics.PullTransportTimeout).Inc()
			if errors.Is(reportCtx.Err(), context.Canceled) {
				// Parent cancel (Sensor stop): do not keep retrying on the short tick.
				return nil, scrapeNonRetryable
			}
			return nil, scrapeRetryable
		}
		log.Warnf("VMScraper: dialing roxagent on %q failed: %v", key, err)
		metrics.PullTransportTotal.WithLabelValues(metrics.PullTransportDialError).Inc()
		return nil, scrapeRetryable
	}
	defer func() { _ = stream.Close() }()

	readStart := s.now()
	result, err := s.client.GetReport(reportCtx, stream, lastKnownToken)
	metrics.PullReadDurationSeconds.Observe(s.now().Sub(readStart).Seconds())
	if err != nil {
		// Classify before cancel(): cancel() looks like Sensor stop to
		// handleGetReportError.
		outcome := s.handleGetReportError(reportCtx, key, err)
		// MAPPING_REQUIRED is the only error whose Meta still needs
		// inspecting: a VM with no mapping at all has nothing to report
		// except this error, and it's the only way Sensor learns to push one.
		if errors.Is(err, vsockclient.ErrMappingRequired) {
			// maybeSyncRepoCPEMapping dials a second connection to the same agent, which
			// enforces at most one connection at a time: this one must be
			// closed first, not left to the deferred close on return.
			_ = stream.Close()
			cancel()
			s.maybeSyncRepoCPEMapping(ctx, vm, key, port, resultMeta(result))
		}
		return nil, outcome
	}
	_ = stream.Close()
	cancel()
	s.maybeSyncRepoCPEMapping(ctx, vm, key, port, result.Meta)
	return result, scrapeOK
}

// resultMeta returns result.Meta, tolerating a nil result: ProtocolClient
// implementations are not required to populate a result on every error.
func resultMeta(result *vsockclient.GetReportResult) *pb.ResponseMeta {
	if result == nil {
		return nil
	}
	return result.Meta
}

func (s *VMScraper) handleGetReportError(ctx context.Context, key string, err error) scrapeOutcome {
	// Timed-out or cancelled reads surface here after Dial's deadline
	// and/or GetReport's close-on-cancel. Prefer ctx.Err() so those land
	// as timeout (matching the dial path) rather than as a protocol/read error.
	if ctx.Err() != nil {
		log.Warnf("VMScraper: reading report from %q timed out: %v", key, err)
		metrics.PullTransportTotal.WithLabelValues(metrics.PullTransportTimeout).Inc()
		if errors.Is(ctx.Err(), context.Canceled) {
			return scrapeNonRetryable
		}
		return scrapeRetryable
	}
	var closeErr closeCoder
	switch {
	case errors.Is(err, vsockclient.ErrNotReady):
		log.Debugf("VMScraper: roxagent on %q has not yet generated a report", key)
		metrics.PullGetReportTotal.WithLabelValues(metrics.PullGetReportNotReady).Inc()
		return scrapeRetryable
	case errors.Is(err, vsockclient.ErrMappingRequired):
		log.Debugf("VMScraper: roxagent on %q has no repository-to-CPE mapping yet", key)
		metrics.PullGetReportTotal.WithLabelValues(metrics.PullGetReportMappingRequired).Inc()
		return scrapeRetryable
	case errors.Is(err, vsockclient.ErrUnknownMethod):
		log.Warnf("VMScraper: roxagent on %q does not support the GetReport method", key)
		metrics.PullGetReportTotal.WithLabelValues(metrics.PullGetReportUnknownMethod).Inc()
		return scrapeNonRetryable
	case errors.Is(err, vsockclient.ErrBusy):
		log.Infof("VMScraper: roxagent on %q is busy with another request: %v", key, err)
		metrics.PullGetReportTotal.WithLabelValues(metrics.PullGetReportBusy).Inc()
		return scrapeRetryable
	case errors.Is(err, vsockclient.ErrInternal):
		log.Warnf("VMScraper: roxagent on %q reported an internal error: %v", key, err)
		metrics.PullGetReportTotal.WithLabelValues(metrics.PullGetReportInternalError).Inc()
		return scrapeRetryable
	case errors.Is(err, vsockclient.ErrMalformedRequest):
		log.Warnf("VMScraper: roxagent on %q rejected the request as malformed: %v", key, err)
		metrics.PullGetReportTotal.WithLabelValues(metrics.PullGetReportMalformedRequest).Inc()
		return scrapeNonRetryable
	case errors.Is(err, vsockclient.ErrRequestTooLarge):
		log.Warnf("VMScraper: roxagent on %q rejected the request as too large: %v", key, err)
		metrics.PullGetReportTotal.WithLabelValues(metrics.PullGetReportRequestTooLarge).Inc()
		return scrapeNonRetryable
	case errors.Is(err, vsockclient.ErrUnknownAgentError):
		log.Warnf("VMScraper: roxagent on %q returned an error code this Sensor version doesn't recognize (possible version mismatch): %v", key, err)
		metrics.PullGetReportTotal.WithLabelValues(metrics.PullGetReportUnknownAgentError).Inc()
		return scrapeRetryable
	case isAbnormalClose(err, &closeErr):
		// e.g. close code 1006, which is what a TLS handshake failure on the
		// agent's side looks like from Sensor's end.
		code, reason := closeErr.CloseCode()
		log.Warnf("VMScraper: roxagent on %q connection closed abnormally (websocket close code %d: %s) — check roxagent's own logs on the VM for the cause",
			key, code, reason)
		metrics.PullTransportTotal.WithLabelValues(metrics.PullTransportAbnormalClose).Inc()
		return scrapeRetryable
	case errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF):
		log.Debugf("VMScraper: roxagent on %q connection closed (agent may be down or restarting): %v", key, err)
		metrics.PullTransportTotal.WithLabelValues(metrics.PullTransportReadError).Inc()
		return scrapeRetryable
	default:
		log.Warnf("VMScraper: unexpected transport or framing error for %q: %v", key, err)
		metrics.PullTransportTotal.WithLabelValues(metrics.PullTransportUnexpected).Inc()
		return scrapeRetryable
	}
}

// closeCodeNormalClosure is the RFC 6455 status code (1000) for a graceful
// websocket close. Declared to keep vmscraper decoupled from the transport
// package.
const closeCodeNormalClosure = 1000

// isAbnormalClose reports a closeCoder whose close code is neither 0 nor
// closeCodeNormalClosure (those fall through to the ordinary io.EOF branch).
func isAbnormalClose(err error, target *closeCoder) bool {
	if !errors.As(err, target) {
		return false
	}
	code, _ := (*target).CloseCode()
	return code != 0 && code != closeCodeNormalClosure
}

// maybeSyncRepoCPEMapping pushes a fresh mapping when the agent is
// Sensor-managed and its reported hash is stale. A URL-managed agent with
// a stale hash is only logged and counted, since Sensor doesn't own it.
func (s *VMScraper) maybeSyncRepoCPEMapping(ctx context.Context, vm *virtualmachine.Info, key string, port uint32, meta *pb.ResponseMeta) {
	updatePath := meta.GetRepoCpeMappingUpdatePath()
	if updatePath == pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_UNSPECIFIED {
		return
	}
	if s.repo2CPEFetcher == nil {
		return
	}
	mapping, sensorHash, ok := s.repo2CPEFetcher.FetchRepo2CPE(ctx)
	if !ok {
		return
	}
	if sensorHash == meta.GetRepoCpeMappingHash() {
		log.Debugf("VMScraper: repo-to-CPE mapping on %q is up to date (hash=%s)", key, sensorHash)
		return
	}

	switch updatePath {
	case pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR:
		s.syncRepoCPEMapping(ctx, vm, key, port, mapping)
	case pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_URL:
		log.Warnf("VMScraper: roxagent on %q reports a URL-managed repo-to-CPE mapping (hash=%s) that differs from Sensor's own cache (hash=%s)",
			key, meta.GetRepoCpeMappingHash(), sensorHash)
		metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncURLHashMismatch).Inc()
	}
}

// syncRepoCPEMapping pushes mapping on a new connection (one request per
// connection). The deadline is independent of GetReport's.
func (s *VMScraper) syncRepoCPEMapping(ctx context.Context, vm *virtualmachine.Info, key string, port uint32, mapping []byte) {
	syncCtx, cancel := context.WithTimeout(ctx, s.perVMTimeout)
	defer cancel()
	if syncCtx.Err() != nil {
		log.Debugf("VMScraper: skipping repo-to-CPE mapping sync for %q: %v", key, syncCtx.Err())
		metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncTimeout).Inc()
		return
	}
	stream, err := s.dialer.Dial(syncCtx, vm.Namespace, vm.Name, port, true)
	if err != nil {
		if syncCtx.Err() != nil {
			log.Warnf("VMScraper: dialing roxagent on %q for repo-to-CPE mapping sync timed out: %v", key, err)
			metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncTimeout).Inc()
			return
		}
		log.Warnf("VMScraper: dialing roxagent on %q for repo-to-CPE mapping sync failed: %v", key, err)
		metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncError).Inc()
		return
	}
	defer func() { _ = stream.Close() }()

	updated, _, err := s.client.SyncRepoCPEMapping(syncCtx, stream, mapping)
	if err != nil {
		// NOT_SENSOR_MANAGED should never happen here: maybeSyncRepoCPEMapping only takes
		// the SENSOR path for agents it believes accept a push. Counting it
		// (without retrying) turns a gating bug into a visible signal.
		if errors.Is(err, vsockclient.ErrMappingNotSensorManaged) {
			log.Warnf("VMScraper: roxagent on %q rejected repo-to-CPE mapping sync as not Sensor-managed", key)
			metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncNotManaged).Inc()
			return
		}
		if syncCtx.Err() != nil {
			log.Warnf("VMScraper: repo-to-CPE mapping sync to %q timed out: %v", key, err)
			metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncTimeout).Inc()
			return
		}
		log.Warnf("VMScraper: repo-to-CPE mapping sync to %q failed: %v", key, err)
		metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncError).Inc()
		return
	}

	log.Infof("VMScraper: synced repo-to-CPE mapping to %q (updated=%t)", key, updated)
	metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncSuccess).Inc()
}

type vmStateSnapshot struct {
	lastToken       string
	lastForwardedAt time.Time
	backoff         time.Duration
}

func (s *VMScraper) snapshotVMState(key string) vmStateSnapshot {
	return concurrency.WithLock1(&s.mu, func() vmStateSnapshot {
		st, ok := s.vmState[key]
		if !ok {
			// reconcile and scrapeKey own vmState membership; a missing key here
			// means the VM was removed concurrently, so return the zero snapshot
			// instead of resurrecting a slot for it.
			return vmStateSnapshot{}
		}
		return vmStateSnapshot{
			lastToken:       st.lastToken,
			lastForwardedAt: st.lastForwardedAt,
			backoff:         st.backoff,
		}
	})
}

// commitVMState records the token from a just-sent report as key's cached scrape state.
//
// Race: If a NACK arrives from Central faster than this function completes, for example, due to:
//   - a scheduling delay on the caller's goroutine between forward returning and this function
//     acquiring `s.mu` (a GC pause, CPU throttling, or GOMAXPROCS contention),
//   - contention on `s.mu` itself, from other concurrent scrapes or NACKs delaying this
//     function's lock acquisition,
//
// then the NACK will overwrite lastToken / backoff / nextAttemptAt, and then this code
// will set the token (and a later scheduleAfterAttempt(scrapeOK) will reset backoff schedule).
// This race is accepted for v1.
func (s *VMScraper) commitVMState(key string, vmID virtualmachine.VMID, newToken string, agentVersion string) {
	concurrency.WithLock(&s.mu, func() {
		state, ok := s.vmState[key]
		if !ok || state.vmID != vmID {
			return
		}
		state.lastToken = newToken
		state.lastForwardedAt = s.now()
		state.lastAgentVersion = agentVersion
	})
}

func (s *VMScraper) forwardReport(ctx context.Context, vm *virtualmachine.Info, report *v4.IndexReport) error {
	if err := s.enqueueToCentral(ctx, newIndexReportMessage(vm, report)); err != nil {
		return err
	}
	metrics.IndexReportsSent.With(metrics.StatusSuccessLabels).Inc()
	return nil
}

func (s *VMScraper) enqueueToCentral(ctx context.Context, msg *message.ExpiringMessage) error {
	if !centralcaps.Has(centralsensor.VirtualMachinesSupported) {
		return errox.NotImplemented.CausedBy(errCapabilityNotSupported)
	}
	if !s.centralReady.IsDone() {
		metrics.IndexReportsSent.With(metrics.StatusCentralNotReadyLabels).Inc()
		return errox.ResourceExhausted.CausedBy(errCentralNotReachable)
	}

	select {
	case <-ctx.Done():
	case s.toCentral <- msg:
		return nil
	default:
		metrics.IndexReportEnqueueBlockedTotal.Inc()
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("waiting to forward: %w", ctx.Err())
	case s.toCentral <- msg:
		return nil
	}
}

func newIndexReportMessage(vm *virtualmachine.Info, report *v4.IndexReport) *message.ExpiringMessage {
	var cidStr string
	if vm.VSOCKCID != nil {
		cidStr = strconv.FormatUint(uint64(*vm.VSOCKCID), 10)
	}
	indexReport := &pb.IndexReport{
		VsockCid: cidStr,
		VmId:     string(vm.ID),
		IndexV4:  report,
	}
	return message.New(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     string(vm.ID),
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachineIndexReport{
					VirtualMachineIndexReport: &pb.IndexReportEvent{
						Id:    string(vm.ID),
						Index: indexReport,
					},
				},
			},
		},
	})
}

func (s *VMScraper) clusterIDString() string {
	if s.clusterID == nil {
		return ""
	}
	return s.clusterID.GetNoWait()
}

func (s *VMScraper) vmUpdateMessage(vm *virtualmachine.Info) *message.ExpiringMessage {
	event := virtualmachine.SensorEvent(central.ResourceAction_UPDATE_RESOURCE, s.clusterIDString(), vm)
	if event == nil {
		return nil
	}
	return message.New(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{Event: event},
	})
}

// tryEnqueueVMUpdate best-effort enqueues a facts UPDATE after a successful
// index forward. It must not block: forwardReport already waited on toCentral,
// and a second blocking send would stall the scrape if the buffer is full.
func (s *VMScraper) tryEnqueueVMUpdate(vm *virtualmachine.Info) {
	if vm == nil || vm.AgentFacts == nil {
		return
	}
	msg := s.vmUpdateMessage(vm)
	if msg == nil {
		return
	}
	select {
	case s.toCentral <- msg:
	default:
	}
}

func (s *VMScraper) persistAgentFacts(vm *virtualmachine.Info, facts map[string]string) {
	mapped, ok := snapshotAgentFacts(facts)
	if !ok {
		return
	}
	vm.AgentFacts = mapped
	// AddOrUpdate stores the pointer it is given. Copy so the scraper's vm
	// is not the store's live object when a later forward copies it unlocked.
	s.store.AddOrUpdate(vm.Copy())
}

// snapshotAgentFacts maps ResponseMeta.facts. ok is false when nothing maps,
// so stored values stay as they are.
func snapshotAgentFacts(facts map[string]string) (mapped map[string]string, ok bool) {
	if len(facts) == 0 {
		return nil, false
	}
	mapped = virtualmachine.AgentFactsFromResponseFacts(facts)
	return mapped, len(mapped) > 0
}

// forwardAgentFactsIfChanged emits a VM update when roxagent facts changed
// even if the index report is unchanged. The store is updated only after
// enqueue succeeds so a failed send is retried on the next unchanged scrape.
func (s *VMScraper) forwardAgentFactsIfChanged(ctx context.Context, vm *virtualmachine.Info, facts map[string]string) {
	mapped, ok := snapshotAgentFacts(facts)
	if !ok {
		return
	}
	logAndRecordDiscoveredFacts(vm.Key(), facts)
	var prevFacts map[string]string
	if prev := s.store.Get(vm.ID); prev != nil {
		prevFacts = prev.AgentFacts
	}
	if maps.Equal(prevFacts, mapped) {
		return
	}
	toSend := vm.Copy()
	toSend.AgentFacts = mapped
	msg := s.vmUpdateMessage(toSend)
	if msg == nil {
		return
	}
	if err := s.enqueueToCentral(ctx, msg); err != nil {
		log.Debugf("VMScraper: agent facts for %q not forwarded; will retry: %v", vm.Key(), err)
		return
	}
	s.store.AddOrUpdate(toSend)
}
