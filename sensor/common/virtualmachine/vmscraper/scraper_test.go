package vmscraper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockStore struct {
	vms []*virtualmachine.Info
}

func (m *mockStore) ListRunning() []*virtualmachine.Info { return m.vms }

type mockDialer struct {
	err      error
	errQueue []error
	callIdx  int
}

func (m *mockDialer) Dial(_ context.Context, _, _ string, _ uint32, _ bool) (io.ReadWriteCloser, error) {
	idx := m.callIdx
	m.callIdx++
	if idx < len(m.errQueue) && m.errQueue[idx] != nil {
		return nil, m.errQueue[idx]
	}
	if m.err != nil {
		return nil, m.err
	}
	return nopCloser{}, nil
}

// blockingDialer waits until ctx is done, then returns ctx.Err(). Used to
// exercise the per-VM timeout classification in dialAndGetReport.
type blockingDialer struct{}

func (blockingDialer) Dial(ctx context.Context, _, _ string, _ uint32, _ bool) (io.ReadWriteCloser, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

type nopCloser struct{}

func (nopCloser) Read([]byte) (int, error)  { return 0, io.EOF }
func (nopCloser) Write([]byte) (int, error) { return 0, nil }
func (nopCloser) Close() error              { return nil }

type mockProtocolClient struct {
	resultQueue []*vsockclient.GetReportResult
	errQueue    []error
	calls       []protocolCall
	callIdx     int

	syncCalls   [][]byte
	syncUpdated bool
	syncMeta    *pb.ResponseMeta
	syncErr     error
}

type protocolCall struct {
	ifNewerThan uint32
	knownEpoch  uint32
}

func (m *mockProtocolClient) GetReport(_ context.Context, _ io.ReadWriteCloser, ifNewerThan uint32, knownEpoch uint32) (*vsockclient.GetReportResult, error) {
	m.calls = append(m.calls, protocolCall{ifNewerThan: ifNewerThan, knownEpoch: knownEpoch})
	idx := m.callIdx
	m.callIdx++
	if idx < len(m.errQueue) && m.errQueue[idx] != nil {
		// Some errors (e.g. MAPPING_REQUIRED) still carry a queued result
		// with Meta, mirroring the real client's error-case behavior.
		var result *vsockclient.GetReportResult
		if idx < len(m.resultQueue) {
			result = m.resultQueue[idx]
		}
		return result, m.errQueue[idx]
	}
	if idx < len(m.resultQueue) {
		return m.resultQueue[idx], nil
	}
	return nil, errors.New("unexpected call: no more queued results")
}

func (m *mockProtocolClient) SyncRepoCPEMapping(_ context.Context, _ io.ReadWriteCloser, mapping []byte) (bool, *pb.ResponseMeta, error) {
	m.syncCalls = append(m.syncCalls, mapping)
	return m.syncUpdated, m.syncMeta, m.syncErr
}

func (m *mockProtocolClient) reset() {
	m.calls = nil
	m.callIdx = 0
	m.syncCalls = nil
}

// fakeFetcher is a Repo2CPEFetcher test double.
type fakeFetcher struct {
	mapping []byte
	hash    string
	ok      bool
	calls   int
}

func (f *fakeFetcher) FetchRepo2CPE(_ context.Context) ([]byte, string, bool) {
	f.calls++
	return f.mapping, f.hash, f.ok
}

type mockSender struct {
	sent []*v4.IndexReport
}

func (m *mockSender) Send(_ context.Context, _ *virtualmachine.Info, report *v4.IndexReport) error {
	m.sent = append(m.sent, report)
	return nil
}

// --- Helpers ---

func ptr32(v uint32) *uint32 { return &v }

func makeVM(ns, name string, cid uint32) *virtualmachine.Info {
	return &virtualmachine.Info{
		Namespace: ns,
		Name:      name,
		VSOCKCID:  ptr32(cid),
		Running:   true,
	}
}

func makeReport(gen uint32) *vsockclient.GetReportResult {
	return &vsockclient.GetReportResult{
		IndexReport: &v4.IndexReport{
			State: "IndexFinished",
		},
		Meta: &pb.ResponseMeta{
			ReportGeneration: gen,
		},
	}
}

func unchangedResult() *vsockclient.GetReportResult {
	return &vsockclient.GetReportResult{
		Unchanged: true,
		Meta:      &pb.ResponseMeta{ReportGeneration: 1},
	}
}

func makeReportWithEpoch(gen, epoch uint32) *vsockclient.GetReportResult {
	return &vsockclient.GetReportResult{
		IndexReport: &v4.IndexReport{
			State: "IndexFinished",
		},
		Meta: &pb.ResponseMeta{
			ReportGeneration: gen,
			Epoch:            epoch,
		},
	}
}

func unchangedResultWithEpoch(gen, epoch uint32) *vsockclient.GetReportResult {
	return &vsockclient.GetReportResult{
		Unchanged: true,
		Meta:      &pb.ResponseMeta{ReportGeneration: gen, Epoch: epoch},
	}
}

func strPtr(s string) *string { return &s }

func metaWithMapping(hash string, path pb.RepoCPEMappingUpdatePath) *pb.ResponseMeta {
	return &pb.ResponseMeta{
		RepoCpeMappingHash:       strPtr(hash),
		RepoCpeMappingUpdatePath: path.Enum(),
	}
}

// --- Tests ---

func TestVMScraper_PollsRunningVMs(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
		makeVM("ns2", "vm-b", 200),
	}}
	sender := &mockSender{}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport(1), makeReport(1)},
		errQueue:    []error{nil, nil},
	}

	s := newTestScraper(store, sender, dialer, client)
	s.pollOnce(context.Background())

	assert.Len(t, sender.sent, 2)
	assert.Len(t, client.calls, 2)
}

func TestVMScraper_SkipsUnchangedGeneration(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	sender := &mockSender{}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport(1)},
	}

	s := newTestScraper(store, sender, dialer, client)

	s.pollOnce(context.Background())
	require.Len(t, sender.sent, 1)

	// Second poll returns unchanged
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResult()}
	s.pollOnce(context.Background())
	assert.Len(t, sender.sent, 1, "should not forward unchanged report")
}

func TestVMScraper_RemainsActiveAcrossUnchangedPolls(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	sender := &mockSender{}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport(1)},
	}

	s := newTestScraper(store, sender, dialer, client)

	s.pollOnce(context.Background())
	require.True(t, s.IsActivelyScraped("ns1/vm-a"))
	require.True(t, s.IsActivelyScraped("100"))

	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResult()}
	s.pollOnce(context.Background())

	assert.Len(t, sender.sent, 1, "should not forward unchanged report")
	assert.True(t, s.IsActivelyScraped("ns1/vm-a"))
	assert.True(t, s.IsActivelyScraped("100"))
}

func TestVMScraper_ForwardsAfter4Hours(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	sender := &mockSender{}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport(1)},
	}

	s := newTestScraper(store, sender, dialer, client)
	s.now = func() time.Time { return time.Now() }

	s.pollOnce(context.Background())
	require.Len(t, sender.sent, 1)

	// Simulate mandatoryRefreshAfter+1s elapsed
	s.now = func() time.Time { return time.Now().Add(s.mandatoryRefreshAfter + time.Second) }

	// The mandatory refresh is known before dialing, so a single call
	// requesting the full report (ifNewerThan=0) is enough.
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{makeReport(1)}
	s.pollOnce(context.Background())

	require.Len(t, client.calls, 1, "mandatory refresh should resolve in a single round trip")
	assert.Equal(t, uint32(0), client.calls[0].ifNewerThan, "mandatory refresh forces the full report on the only call")
	assert.Equal(t, uint32(0), client.calls[0].knownEpoch, "mandatory refresh forces the full report on the only call")
	assert.Len(t, sender.sent, 2, "should forward after 4h even if unchanged")
}

// TestVMScraper_ForwardsOnEpochMismatch reproduces the restart-collision
// scenario: roxagent restarts and its report_generation counter (which resets
// to 1 on every restart) coincidentally climbs back to the exact value Sensor
// already has cached for this VM. This test exercises the client-side
// fallback path (mockProtocolClient returns Unchanged with a mismatched
// epoch, as an older roxagent that ignores knownEpoch would); a current
// roxagent instead resolves this in the first round trip (see
// vsockserver.protocol_test.go on the roxagent-serve branch).
func TestVMScraper_ForwardsOnEpochMismatch(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	sender := &mockSender{}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReportWithEpoch(5, 100)},
	}

	s := newTestScraper(store, sender, dialer, client)
	s.pollOnce(context.Background())
	require.Len(t, sender.sent, 1)

	// Agent restarts and, before Sensor's next poll, coincidentally climbs back
	// to generation=5 with a new epoch. roxagent's own generation-only check
	// reports "unchanged"; Sensor must not trust that because the epoch moved.
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{
		unchangedResultWithEpoch(5, 200),
		makeReportWithEpoch(5, 200),
	}
	s.pollOnce(context.Background())

	require.Len(t, client.calls, 2, "epoch mismatch should force a second call for the full report")
	assert.Equal(t, uint32(5), client.calls[0].ifNewerThan, "first call uses last generation")
	assert.Equal(t, uint32(100), client.calls[0].knownEpoch, "first call sends the previously cached epoch")
	assert.Equal(t, uint32(0), client.calls[1].ifNewerThan, "second call forces full report")
	assert.Equal(t, uint32(0), client.calls[1].knownEpoch, "second (forced) call doesn't need to send an epoch")
	assert.Len(t, sender.sent, 2, "should forward despite matching generation, because epoch changed")
}

// TestVMScraper_TrustsUnchangedWhenEpochZero ensures backward compatibility
// with roxagent builds that predate the epoch field (always reporting epoch
// 0): Sensor must keep trusting the generation-only Unchanged flag as before,
// not treat every poll as a restart.
func TestVMScraper_TrustsUnchangedWhenEpochZero(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	sender := &mockSender{}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReportWithEpoch(1, 0)},
	}

	s := newTestScraper(store, sender, dialer, client)
	s.pollOnce(context.Background())
	require.Len(t, sender.sent, 1)

	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResultWithEpoch(1, 0)}
	s.pollOnce(context.Background())

	require.Len(t, client.calls, 1, "epoch 0 means the agent predates the field; no forced second call")
	assert.Len(t, sender.sent, 1, "should not forward unchanged report from a pre-epoch agent")
}

// TestVMScraper_SendsKnownEpochOnRequest verifies Sensor sends its
// last-cached epoch on every request, letting a current roxagent resolve a
// restart-coincidence false match in a single round trip instead of relying
// on the client-side fallback.
func TestVMScraper_SendsKnownEpochOnRequest(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	sender := &mockSender{}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReportWithEpoch(1, 100)},
	}

	s := newTestScraper(store, sender, dialer, client)
	s.pollOnce(context.Background())

	require.Len(t, client.calls, 1)
	assert.Equal(t, uint32(0), client.calls[0].knownEpoch, "first-ever request for a VM has no cached epoch")

	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResultWithEpoch(1, 100)}
	s.pollOnce(context.Background())

	require.Len(t, client.calls, 1, "matching epoch and generation should resolve in a single round trip")
	assert.Equal(t, uint32(100), client.calls[0].knownEpoch, "subsequent requests send the cached epoch")
}

func TestVMScraper_ForwardsOnGenerationChange(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	sender := &mockSender{}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport(1)},
	}

	s := newTestScraper(store, sender, dialer, client)
	s.pollOnce(context.Background())
	require.Len(t, sender.sent, 1)

	// New generation
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{makeReport(2)}
	s.pollOnce(context.Background())
	assert.Len(t, sender.sent, 2, "should forward on generation change")
}

func TestVMScraper_HandlesDialAndProtocolFailures(t *testing.T) {
	vms := []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
		makeVM("ns2", "vm-b", 200),
	}

	cases := map[string]struct {
		dialer       VMDialer
		perVMTimeout time.Duration
		resultQueue  []*vsockclient.GetReportResult
		errQueue     []error
		wantCalls    int
		wantSent     int
	}{
		"should still send for vm-b when vm-a hits a protocol error": {
			dialer:      &mockDialer{},
			resultQueue: []*vsockclient.GetReportResult{nil, makeReport(1)},
			errQueue:    []error{errors.New("connection refused"), nil},
			wantCalls:   2,
			wantSent:    1,
		},
		"should still send for vm-b when vm-a dial fails": {
			dialer: &mockDialer{
				errQueue: []error{errors.New("dial failed"), nil},
			},
			resultQueue: []*vsockclient.GetReportResult{makeReport(1)},
			wantCalls:   1,
			wantSent:    1,
		},
		"should forward nothing when every dial times out": {
			dialer:       blockingDialer{},
			perVMTimeout: 20 * time.Millisecond,
			wantCalls:    0,
			wantSent:     0,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			sender := &mockSender{}
			client := &mockProtocolClient{
				resultQueue: tc.resultQueue,
				errQueue:    tc.errQueue,
			}
			s := newTestScraper(&mockStore{vms: vms}, sender, tc.dialer, client)
			if tc.perVMTimeout > 0 {
				s.perVMTimeout = tc.perVMTimeout
			}
			s.pollOnce(context.Background())

			assert.Len(t, client.calls, tc.wantCalls)
			assert.Len(t, sender.sent, tc.wantSent)
		})
	}
}

func TestVMScraper_StartRejectsSecondCall(t *testing.T) {
	s := newTestScraper(&mockStore{}, &mockSender{}, &mockDialer{}, &mockProtocolClient{})
	require.NoError(t, s.Start())
	t.Cleanup(s.Stop)

	assert.ErrorIs(t, s.Start(), errStartMoreThanOnce)
}

// GetReport does not take a context; Dial stamps ctx's deadline onto the
// socket, so a timed-out read returns a plain I/O error after ctx is already
// done. Classification must follow ctx.Err(), same as the dial-failure path.
func TestVMScraper_GetReportTimeoutClassified(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)
	client := &mockProtocolClient{
		errQueue: []error{errors.New("i/o timeout")},
	}
	s := newTestScraper(&mockStore{vms: []*virtualmachine.Info{vm}}, &mockSender{}, &mockDialer{}, client)

	timeoutBefore := testutil.ToFloat64(metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusTimeout))
	readErrBefore := testutil.ToFloat64(metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusReadError))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, ok := s.dialAndGetReport(ctx, vm, "ns1/vm-a", 1, 0, 0)

	assert.False(t, ok)
	assert.Equal(t, timeoutBefore+1, testutil.ToFloat64(metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusTimeout)))
	assert.Equal(t, readErrBefore, testutil.ToFloat64(metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusReadError)),
		"timed-out read must not be counted as a protocol/read error")
}

func TestVMScraper_PrunesStaleState(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
		makeVM("ns2", "vm-b", 200),
	}}
	sender := &mockSender{}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport(1), makeReport(1)},
	}

	s := newTestScraper(store, sender, dialer, client)
	s.pollOnce(context.Background())
	assert.Len(t, s.vmState, 2)
	assert.True(t, s.activeVMs.Contains("ns1/vm-a"))

	// Remove vm-a from running set
	store.vms = []*virtualmachine.Info{makeVM("ns2", "vm-b", 200)}
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{makeReport(2)}
	s.pollOnce(context.Background())

	assert.Len(t, s.vmState, 1, "stale vm-a state should be pruned")
	assert.False(t, s.activeVMs.Contains("ns1/vm-a"), "vm-a should no longer be active")
	assert.True(t, s.activeVMs.Contains("ns2/vm-b"))
}

func newTestScraper(store RunningVMStore, sender IndexReportSender, dialer VMDialer, client ProtocolClient) *VMScraper {
	return &VMScraper{
		store:                 store,
		sender:                sender,
		dialer:                dialer,
		client:                client,
		interval:              5 * time.Minute,
		perVMTimeout:          10 * time.Second,
		mandatoryRefreshAfter: 4 * time.Hour,
		concurrency:           1,
		// Half of the 16MiB default pull response-size ceiling — same
		// derivation New() uses from env.VirtualMachinesPullMaxResponseSizeKB.
		warnMaxBytes: 8 << 20,
		vmState:      make(map[string]*vmState),
		activeVMs:    set.NewStringSet(),
		now:          time.Now,
	}
}

// --- Thread-safe mocks for concurrent tests ---

// delayDialer sleeps for delay on every Dial and tracks the highest number
// of Dial calls observed in flight at once, so tests can assert on actual
// concurrency instead of inferring it from wall-clock elapsed time (which is
// prone to flaking under CI load/GC pauses).
type delayDialer struct {
	delay       time.Duration
	inFlight    atomic.Int32
	maxObserved atomic.Int32
}

func (d *delayDialer) Dial(_ context.Context, _, _ string, _ uint32, _ bool) (io.ReadWriteCloser, error) {
	cur := d.inFlight.Add(1)
	defer d.inFlight.Add(-1)
	for {
		prevMax := d.maxObserved.Load()
		if cur <= prevMax || d.maxObserved.CompareAndSwap(prevMax, cur) {
			break
		}
	}
	time.Sleep(d.delay)
	return nopCloser{}, nil
}

type safeProtocolClient struct {
	mu    sync.Mutex
	gen   uint32
	calls int
}

func (c *safeProtocolClient) GetReport(_ context.Context, _ io.ReadWriteCloser, _ uint32, _ uint32) (*vsockclient.GetReportResult, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return makeReport(c.gen), nil
}

func (c *safeProtocolClient) SyncRepoCPEMapping(_ context.Context, _ io.ReadWriteCloser, _ []byte) (bool, *pb.ResponseMeta, error) {
	return false, nil, nil
}

type safeSender struct {
	mu   sync.Mutex
	sent int
}

func (s *safeSender) Send(_ context.Context, _ *virtualmachine.Info, _ *v4.IndexReport) error {
	s.mu.Lock()
	s.sent++
	s.mu.Unlock()
	return nil
}

func TestVMScraper_ConcurrentFasterThanSequential(t *testing.T) {
	const (
		numVMs      = 10
		dialDelay   = 50 * time.Millisecond
		concurrency = 10
	)

	vms := make([]*virtualmachine.Info, numVMs)
	for i := range vms {
		vms[i] = makeVM("ns", fmt.Sprintf("vm-%d", i), uint32(100+i))
	}

	store := &mockStore{vms: vms}
	sender := &safeSender{}
	dialer := &delayDialer{delay: dialDelay}
	client := &safeProtocolClient{gen: 1}

	s := &VMScraper{
		store:        store,
		sender:       sender,
		dialer:       dialer,
		client:       client,
		interval:     5 * time.Minute,
		perVMTimeout: 10 * time.Second,
		concurrency:  concurrency,
		warnMaxBytes: 8 << 20,
		vmState:      make(map[string]*vmState),
		activeVMs:    set.NewStringSet(),
		now:          time.Now,
	}

	s.pollOnce(context.Background())

	assert.Equal(t, numVMs, sender.sent)
	assert.Equal(t, numVMs, client.calls)
	// Direct concurrency signal instead of a wall-clock inference: with
	// concurrency == numVMs, all dials should be in flight together at some
	// point during the cycle. A threshold below numVMs still proves
	// concurrent (rather than sequential) execution without being flaky
	// about exactly how many overlap under scheduler/CI load.
	maxObserved := dialer.maxObserved.Load()
	require.Greater(t, maxObserved, int32(1),
		"expected multiple VMs to be dialed concurrently, observed max concurrency of %d", maxObserved)
}

func TestVMScraper_MaybeSync(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)

	cases := map[string]struct {
		meta          *pb.ResponseMeta
		fetcher       *fakeFetcher
		noFetcher     bool
		syncErr       error
		wantSyncCalls int
		wantMetric    string
	}{
		"hash match on the SENSOR path should not dial a second time": {
			meta:    metaWithMapping("same-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR),
			fetcher: &fakeFetcher{ok: true, hash: "same-hash"},
		},
		"SENSOR mismatch should dial and sync": {
			meta:          metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR),
			fetcher:       &fakeFetcher{ok: true, hash: "new-hash", mapping: []byte("payload")},
			wantSyncCalls: 1,
			wantMetric:    metrics.PullStatusSyncSuccess,
		},
		"URL mismatch should log and count but never dial": {
			meta:       metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_URL),
			fetcher:    &fakeFetcher{ok: true, hash: "new-hash"},
			wantMetric: metrics.PullStatusURLHashMismatch,
		},
		"UNSPECIFIED update path should never sync": {
			meta:    metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_UNSPECIFIED),
			fetcher: &fakeFetcher{ok: true, hash: "new-hash"},
		},
		"nil meta (agent predates this feature) should never sync": {
			meta:    nil,
			fetcher: &fakeFetcher{ok: true, hash: "new-hash"},
		},
		"fetcher not ok should never sync even on a SENSOR mismatch": {
			meta:    metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR),
			fetcher: &fakeFetcher{ok: false},
		},
		"no fetcher ever set should be a no-op": {
			meta:      metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR),
			noFetcher: true,
		},
		"NOT_SENSOR_MANAGED on the sync dial should log and count, not retry": {
			meta:          metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR),
			fetcher:       &fakeFetcher{ok: true, hash: "new-hash", mapping: []byte("payload")},
			syncErr:       vsockclient.ErrMappingNotSensorManaged,
			wantSyncCalls: 1,
			wantMetric:    metrics.PullStatusSyncNotManaged,
		},
		"a generic sync failure should log and count": {
			meta:          metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR),
			fetcher:       &fakeFetcher{ok: true, hash: "new-hash", mapping: []byte("payload")},
			syncErr:       errors.New("connection reset"),
			wantSyncCalls: 1,
			wantMetric:    metrics.PullStatusSyncError,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			client := &mockProtocolClient{syncErr: tc.syncErr}
			s := newTestScraper(&mockStore{}, &mockSender{}, &mockDialer{}, client)
			if !tc.noFetcher {
				s.SetRepo2CPEFetcher(tc.fetcher)
			}

			var before float64
			if tc.wantMetric != "" {
				before = testutil.ToFloat64(metrics.PullRequestsTotal.WithLabelValues(tc.wantMetric))
			}

			s.maybeSync(context.Background(), vm, "ns1/vm-a", 9999, tc.meta)

			assert.Len(t, client.syncCalls, tc.wantSyncCalls)
			if tc.wantMetric != "" {
				assert.Equal(t, before+1, testutil.ToFloat64(metrics.PullRequestsTotal.WithLabelValues(tc.wantMetric)))
			}
		})
	}
}

// TestVMScraper_DialAndGetReport_SyncTriggering exercises maybeSync's
// integration point inside dialAndGetReport: only a successful exchange or
// MAPPING_REQUIRED carries usable Meta, so only those should ever reach the
// fetcher/second dial.
func TestVMScraper_DialAndGetReport_SyncTriggering(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)
	staleFetcher := func() *fakeFetcher {
		return &fakeFetcher{ok: true, hash: "new-hash", mapping: []byte("payload")}
	}

	cases := map[string]struct {
		resultQueue   []*vsockclient.GetReportResult
		errQueue      []error
		wantOK        bool
		wantSyncCalls int
	}{
		"MAPPING_REQUIRED error triggers a sync attempt": {
			resultQueue:   []*vsockclient.GetReportResult{{Meta: metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR)}},
			errQueue:      []error{vsockclient.ErrMappingRequired},
			wantOK:        false,
			wantSyncCalls: 1,
		},
		"NOT_READY never triggers a sync attempt": {
			errQueue: []error{vsockclient.ErrNotReady},
			wantOK:   false,
		},
		"a generic protocol error never triggers a sync attempt": {
			errQueue: []error{errors.New("connection refused")},
			wantOK:   false,
		},
		"a successful report with a stale Sensor-managed hash also triggers a sync": {
			resultQueue: []*vsockclient.GetReportResult{{
				IndexReport: &v4.IndexReport{State: "IndexFinished"},
				Meta:        metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR),
			}},
			wantOK:        true,
			wantSyncCalls: 1,
		},
		"an Unchanged report with a matching hash never triggers a sync": {
			resultQueue: []*vsockclient.GetReportResult{{
				Unchanged: true,
				Meta:      metaWithMapping("new-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR),
			}},
			wantOK: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			client := &mockProtocolClient{resultQueue: tc.resultQueue, errQueue: tc.errQueue}
			s := newTestScraper(&mockStore{}, &mockSender{}, &mockDialer{}, client)
			s.SetRepo2CPEFetcher(staleFetcher())

			_, ok := s.dialAndGetReport(context.Background(), vm, "ns1/vm-a", 1, 0, 0)

			assert.Equal(t, tc.wantOK, ok)
			assert.Len(t, client.syncCalls, tc.wantSyncCalls)
		})
	}
}

func TestClampPollInterval(t *testing.T) {
	cases := map[string]struct {
		in   time.Duration
		want time.Duration
	}{
		"should leave values at the minimum unchanged": {
			in:   time.Minute,
			want: time.Minute,
		},
		"should leave values above the minimum unchanged": {
			in:   5 * time.Minute,
			want: 5 * time.Minute,
		},
		"should clamp values below the minimum up to one minute": {
			in:   30 * time.Second,
			want: time.Minute,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, clampPollInterval(tc.in))
		})
	}
}
