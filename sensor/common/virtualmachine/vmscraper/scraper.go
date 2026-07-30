package vmscraper

import (
	"context"
	"errors"
	"io"
	"strconv"
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

const (
	mandatoryRefreshAfter = 4 * time.Hour
)

func getVsockPort() uint32 {
	return uint32(env.VirtualMachinesVsockPort.IntegerSetting())
}

// RunningVMStore provides the list of running VMs.
type RunningVMStore interface {
	ListRunning() []*virtualmachine.Info
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
	GetReport(stream io.ReadWriteCloser, ifNewerThan uint32, knownEpoch uint32) (*vsockclient.GetReportResult, error)
}

// VMScraper polls running VMs and pulls their scan reports via VSOCK.
type VMScraper struct {
	store        RunningVMStore
	sender       IndexReportSender
	dialer       VMDialer
	client       ProtocolClient
	interval     time.Duration
	perVMTimeout time.Duration
	concurrency  int
	warnMaxBytes int
	stopper      concurrency.Stopper
	started      atomic.Bool
	now          func() time.Time

	mu        sync.Mutex
	vmState   map[string]*vmState
	activeVMs set.StringSet
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
		store:        store,
		sender:       sender,
		dialer:       dialer,
		client:       client,
		interval:     env.VirtualMachinesScraperPollInterval.DurationSetting(),
		perVMTimeout: env.VirtualMachinesScraperPerVMTimeout.DurationSetting(),
		concurrency:  env.VirtualMachinesScraperConcurrency.IntegerSetting(),
		// Warn once a report is halfway to the hard response-size ceiling,
		// so operators get advance notice before reports start actually
		// being rejected at that limit.
		warnMaxBytes: env.VirtualMachinesPullMaxResponseSizeKB.IntegerSetting() * 1024 / 2,
		vmState:      make(map[string]*vmState),
		activeVMs:    set.NewStringSet(),
		now:          time.Now,
	}
}

// IsActivelyScraped reports whether the given VM is actively scraped via pull
// mode. Accepts either a "namespace/name" key or a vsock CID string.
// Used to suppress duplicate push-mode reports during the push→pull transition.
func (s *VMScraper) IsActivelyScraped(key string) bool {
	return concurrency.WithLock1(&s.mu, func() bool {
		return s.activeVMs.Contains(key)
	})
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
func (s *VMScraper) ProcessMessage(_ context.Context, _ *central.MsgToSensor) error {
	return nil
}
func (s *VMScraper) Accepts(_ *central.MsgToSensor) bool         { return false }
func (s *VMScraper) ResponsesC() <-chan *message.ExpiringMessage { return nil }

func (s *VMScraper) run() {
	defer s.stopper.Flow().ReportStopped()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-s.stopper.Flow().StopRequested()
		cancel()
	}()

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

	// Build a new activeVMs set during the cycle. The old set remains in
	// effect for IsActivelyScraped callers, preventing a suppression gap
	// while scraping is in progress.
	scrapedVMs := set.NewStringSet()
	var successCount atomic.Int32

	liveKeys := set.NewStringSet()
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(s.concurrency)

	for _, vm := range vms {
		liveKeys.Add(vmKey(vm))
		g.Go(func() error {
			if s.scrapeVM(gCtx, vm, scrapedVMs) {
				successCount.Add(1)
			}
			return nil
		})
	}
	_ = g.Wait()

	concurrency.WithLock(&s.mu, func() {
		s.activeVMs = scrapedVMs
	})

	s.pruneStaleVMState(liveKeys)
	elapsed := time.Since(cycleStart)
	log.Infof("VMScraper: cycle done: %d/%d VMs scraped successfully in %s", successCount.Load(), len(vms), elapsed.Truncate(time.Millisecond))
	metrics.PullCycleDurationSeconds.Observe(elapsed.Seconds())
}

func (s *VMScraper) scrapeVM(ctx context.Context, vm *virtualmachine.Info, scrapedVMs set.StringSet) bool {
	key := vmKey(vm)
	state := s.getOrCreateState(key)

	vmCtx, cancel := context.WithTimeout(ctx, s.perVMTimeout)
	defer cancel()

	totalStart := s.now()
	port := getVsockPort()

	// The mandatory refresh is a deterministic, time-based decision Sensor
	// can make before dialing at all, so request the full report on this
	// first (and only) round trip instead of asking "anything newer?" and
	// re-dialing once told no.
	ifNewerThan, knownEpoch := state.lastGeneration, state.lastEpoch
	mandatoryRefreshDue := s.now().Sub(state.lastForwardedAt) > mandatoryRefreshAfter
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
		epochMismatch := epoch != 0 && epoch != state.lastEpoch
		if !epochMismatch {
			s.registerScrapedVM(scrapedVMs, vm)
			log.Debugf("VMScraper: unchanged report from roxagent on %q (generation=%d)", key, state.lastGeneration)
			metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusUnchanged).Inc()
			return true
		}
		log.Infof("VMScraper: roxagent on %q restarted (epoch changed from %d to %d, generation coincidentally matched cached value %d) — forcing full report",
			key, state.lastEpoch, epoch, state.lastGeneration)
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

	if err := s.sender.Send(vmCtx, vm, result.IndexReport); err != nil {
		log.Errorf("VMScraper: sending %q report to Central failed: %v", key, err)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusSendError).Inc()
		return false
	}

	state.lastGeneration = result.Meta.GetReportGeneration()
	state.lastEpoch = result.Meta.GetEpoch()
	state.lastForwardedAt = s.now()
	s.registerScrapedVM(scrapedVMs, vm)

	log.Debugf("VMScraper: successfully pulled report for %q: generation=%d, packages=%d, size=%d bytes, total=%s",
		key, state.lastGeneration, len(result.IndexReport.GetContents().GetPackages()), reportSize,
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
	result, err := s.client.GetReport(stream, ifNewerThan, knownEpoch)
	metrics.PullReadDurationSeconds.Observe(time.Since(readStart).Seconds())
	if err != nil {
		s.handleGetReportError(key, err)
		return nil, false
	}
	return result, true
}

func (s *VMScraper) handleGetReportError(key string, err error) {
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
	default:
		log.Warnf("VMScraper: protocol error for %q (possible version mismatch): %v", key, err)
		metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusReadError).Inc()
	}
}

// vmKey returns the identifier used for vmState and activeVMs lookups.
func vmKey(vm *virtualmachine.Info) string {
	return vm.Namespace + "/" + vm.Name
}

func (s *VMScraper) registerScrapedVM(scrapedVMs set.StringSet, vm *virtualmachine.Info) {
	key := vmKey(vm)
	// Register both identifiers so IsActivelyScraped works whether the
	// caller uses "namespace/name" (pull path) or a CID string (push path).
	concurrency.WithLock(&s.mu, func() {
		scrapedVMs.Add(key)
		if vm.VSOCKCID != nil {
			scrapedVMs.Add(strconv.FormatUint(uint64(*vm.VSOCKCID), 10))
		}
	})
}

// getOrCreateState returns the vmState for key, creating it if absent.
// The returned pointer is mutated outside the lock — this is safe because
// each VM key is processed by exactly one goroutine per poll cycle
// (the VM list contains unique entries, and errgroup assigns one goroutine each).
func (s *VMScraper) getOrCreateState(key string) *vmState {
	return concurrency.WithLock1(&s.mu, func() *vmState {
		st, ok := s.vmState[key]
		if !ok {
			st = &vmState{}
			s.vmState[key] = st
		}
		return st
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
