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
	"github.com/stackrox/rox/generated/internalapi/central"
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
		return nil, m.errQueue[idx]
	}
	if idx < len(m.resultQueue) {
		return m.resultQueue[idx], nil
	}
	return nil, errors.New("unexpected call: no more queued results")
}

func (m *mockProtocolClient) reset() {
	m.calls = nil
	m.callIdx = 0
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

// TestVMScraper_NACK reproduces a scenario where Central NACKs a report
// (e.g. Scanner was still starting up). Without resetting the cached
// generation, the next poll would see roxagent report "unchanged"
// (report_generation didn't change) and skip resending, stranding the VM
// until mandatoryRefreshAfter (4h) instead of retrying on the next poll
// interval. It also verifies a NACK for an unrelated VM ID leaves the known
// VM's state untouched instead of forcing a spurious resend.
func TestVMScraper_NACK(t *testing.T) {
	cases := map[string]struct {
		nackResourceID             string
		pollResultAfterNack        *vsockclient.GetReportResult
		wantIfNewerThanOnRetryPoll uint32
		wantTotalSent              int
	}{
		"resets generation and forces an immediate resend when NACK matches the running VM": {
			nackResourceID:             "vm-a-id:100",
			pollResultAfterNack:        makeReport(1),
			wantIfNewerThanOnRetryPoll: 0, // on the second poll, the scraper should ask for a full report
			wantTotalSent:              2,
		},
		"is a no-op when NACK references an unrelated VM ID": {
			nackResourceID:             "unknown-vm-id:999",
			pollResultAfterNack:        unchangedResult(),
			wantIfNewerThanOnRetryPoll: 1, // on the second poll, the scraper should keep trusting the old generation
			wantTotalSent:              1,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vmA := makeVM("ns1", "vm-a", 100)
			vmA.ID = "vm-a-id"
			store := &mockStore{vms: []*virtualmachine.Info{vmA}}
			sender := &mockSender{}
			dialer := &mockDialer{}
			client := &mockProtocolClient{
				resultQueue: []*vsockclient.GetReportResult{makeReport(1)},
			}

			s := newTestScraper(store, sender, dialer, client)
			s.pollOnce(context.Background())
			require.Len(t, sender.sent, 1)

			err := s.ProcessMessage(context.Background(), &central.MsgToSensor{
				Msg: &central.MsgToSensor_SensorAck{
					SensorAck: &central.SensorACK{
						MessageType: central.SensorACK_VM_INDEX_REPORT,
						Action:      central.SensorACK_NACK,
						ResourceId:  tc.nackResourceID,
					},
				},
			})
			require.NoError(t, err)

			client.reset()
			client.resultQueue = []*vsockclient.GetReportResult{tc.pollResultAfterNack}
			s.pollOnce(context.Background())

			require.Len(t, client.calls, 1)
			assert.Equal(t, tc.wantIfNewerThanOnRetryPoll, client.calls[0].ifNewerThan,
				"generation the scraper requests on the poll following the NACK")
			assert.Len(t, sender.sent, tc.wantTotalSent, "total reports handed to the sender across both polls")
		})
	}
}

// gateSender blocks the Nth Send (1-based) until release is closed.
type gateSender struct {
	blockAt int
	entered chan struct{}
	release chan struct{}
	mu      sync.Mutex
	n       int
	sent    []*v4.IndexReport
}

func (g *gateSender) Send(_ context.Context, _ *virtualmachine.Info, report *v4.IndexReport) error {
	g.mu.Lock()
	g.n++
	n := g.n
	g.sent = append(g.sent, report)
	g.mu.Unlock()
	if n == g.blockAt {
		close(g.entered)
		<-g.release
	}
	return nil
}

func TestVMScraper_NACKWinsOverInFlightSend(t *testing.T) {
	vmA := makeVM("ns1", "vm-a", 100)
	vmA.ID = "vm-a-id"
	store := &mockStore{vms: []*virtualmachine.Info{vmA}}
	client := &mockProtocolClient{resultQueue: []*vsockclient.GetReportResult{makeReport(1)}}
	s := newTestScraper(store, &mockSender{}, &mockDialer{}, client)

	s.pollOnce(t.Context())
	require.Equal(t, uint32(1), s.vmState["ns1/vm-a"].lastGeneration)

	gate := &gateSender{
		blockAt: 1,
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	s.sender = gate
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{makeReport(1)}

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.pollOnce(t.Context())
	}()

	select {
	case <-gate.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for in-flight Send")
	}

	require.NoError(t, s.ProcessMessage(t.Context(), &central.MsgToSensor{
		Msg: &central.MsgToSensor_SensorAck{
			SensorAck: &central.SensorACK{
				MessageType: central.SensorACK_VM_INDEX_REPORT,
				Action:      central.SensorACK_NACK,
				ResourceId:  "vm-a-id:100",
			},
		},
	}))
	assert.Equal(t, uint32(0), s.vmState["ns1/vm-a"].lastGeneration)

	close(gate.release)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for pollOnce to finish")
	}
	assert.Equal(t, uint32(0), s.vmState["ns1/vm-a"].lastGeneration,
		"in-flight Send must not overwrite a concurrent NACK reset")

	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{makeReport(1)}
	s.sender = &mockSender{}
	s.pollOnce(t.Context())
	require.Len(t, client.calls, 1)
	assert.Equal(t, uint32(0), client.calls[0].ifNewerThan)
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
