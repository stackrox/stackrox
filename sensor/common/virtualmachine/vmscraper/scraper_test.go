package vmscraper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/concurrency"
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
	vms              []*virtualmachine.Info
	listRunningCalls int
}

// ListRunning mirrors the production store's contract of only returning VMs
// with Running set, so tests exercise reconcile pruning rather than the
// scrapeKey nil-VM guard when a VM stops running.
func (m *mockStore) ListRunning() []*virtualmachine.Info {
	m.listRunningCalls++
	var out []*virtualmachine.Info
	for _, vm := range m.vms {
		if vm.Running {
			out = append(out, vm)
		}
	}
	return out
}

func (m *mockStore) Get(id virtualmachine.VMID) *virtualmachine.Info {
	for _, vm := range m.vms {
		if vm.ID == id {
			return vm
		}
	}
	return nil
}

type mockDialer struct {
	mu       sync.Mutex
	err      error
	errQueue []error
	callIdx  int
}

func (m *mockDialer) Dial(_ context.Context, _, _ string, _ uint32, _ bool) (io.ReadWriteCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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
	mu          sync.Mutex
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
	m.mu.Lock()
	defer m.mu.Unlock()
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
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = nil
	m.callIdx = 0
}

type mockSender struct {
	mu   sync.Mutex
	sent []*v4.IndexReport
	err  error
}

func (m *mockSender) Send(_ context.Context, _ *virtualmachine.Info, report *v4.IndexReport) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, report)
	return nil
}

// --- Helpers ---

func makeVM(ns, name string, cid uint32) *virtualmachine.Info {
	return &virtualmachine.Info{
		ID:        virtualmachine.VMID(ns + "/" + name),
		Namespace: ns,
		Name:      name,
		VSOCKCID:  new(cid),
		Running:   true,
	}
}

// testClock is a mutable clock for schedule/backoff tests.
type testClock struct {
	mu sync.Mutex
	t  time.Time
}

func newTestClock() *testClock {
	return &testClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func makeReport(gen uint32) *vsockclient.GetReportResult {
	return &vsockclient.GetReportResult{
		IndexReport: &v4.IndexReport{
			State: "IndexFinished",
		},
		Meta: &pb.ResponseMeta{
			ReportGeneration: gen,
			Facts: map[string]string{
				"detected_os":         "RHEL",
				"activation_status":   "ACTIVE",
				"dnf_metadata_status": "AVAILABLE",
			},
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

	s, _ := newTestScraper(store, sender, dialer, client)
	discoveredBefore := testutil.ToFloat64(metrics.VMDiscoveredData.WithLabelValues("RHEL", "ACTIVE", "AVAILABLE"))
	s.pollOnce(context.Background())

	assert.Len(t, sender.sent, 2)
	assert.Len(t, client.calls, 2)
	assert.Equal(t, discoveredBefore+2, testutil.ToFloat64(metrics.VMDiscoveredData.WithLabelValues("RHEL", "ACTIVE", "AVAILABLE")))
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

	s, clock := newTestScraper(store, sender, dialer, client)

	s.pollOnce(context.Background())
	require.Len(t, sender.sent, 1)

	// Second poll returns unchanged
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResult()}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())
	assert.Len(t, sender.sent, 1, "should not forward unchanged report")
}

func TestVMScraper_RemainsScheduledAcrossUnchangedPolls(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	sender := &mockSender{}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport(1)},
	}

	s, clock := newTestScraper(store, sender, dialer, client)

	s.pollOnce(context.Background())
	require.True(t, hasScheduleSlot(t, s, "ns1/vm-a"))

	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResult()}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())

	assert.Len(t, sender.sent, 1, "should not forward unchanged report")
	assert.True(t, hasScheduleSlot(t, s, "ns1/vm-a"))
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

	s, clock := newTestScraper(store, sender, dialer, client)

	s.pollOnce(context.Background())
	require.Len(t, sender.sent, 1)

	// Past both the success schedule and the mandatory refresh window.
	clock.Advance(s.mandatoryRefreshAfter + time.Second)

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

// TestVMScraper_ForwardsOnEpochChangeInSingleDial covers a current agent that
// resolves restart-coincidence in one response (full report, new epoch).
func TestVMScraper_ForwardsOnEpochChangeInSingleDial(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	sender := &mockSender{}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReportWithEpoch(5, 100)},
	}

	s, clock := newTestScraper(store, sender, dialer, client)
	s.pollOnce(context.Background())
	require.Len(t, sender.sent, 1)

	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{makeReportWithEpoch(5, 200)}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())

	require.Len(t, client.calls, 1, "current agent serves the full report in one dial")
	assert.Equal(t, uint32(5), client.calls[0].ifNewerThan)
	assert.Equal(t, uint32(100), client.calls[0].knownEpoch)
	assert.Len(t, sender.sent, 2)
}

// TestVMScraper_SendsKnownEpochOnRequest verifies Sensor sends its
// last-cached epoch on every request, letting a current roxagent resolve a
// restart-coincidence false match in a single round trip.
func TestVMScraper_SendsKnownEpochOnRequest(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	sender := &mockSender{}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReportWithEpoch(1, 100)},
	}

	s, clock := newTestScraper(store, sender, dialer, client)
	s.pollOnce(context.Background())

	require.Len(t, client.calls, 1)
	assert.Equal(t, uint32(0), client.calls[0].knownEpoch, "first-ever request for a VM has no cached epoch")

	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResultWithEpoch(1, 100)}
	clock.Advance(s.interval)
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

	s, clock := newTestScraper(store, sender, dialer, client)
	s.pollOnce(context.Background())
	require.Len(t, sender.sent, 1)

	// New generation
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{makeReport(2)}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())
	assert.Len(t, sender.sent, 2, "should forward on generation change")
}

// TestVMScraper_NACK reproduces a scenario where Central NACKs a report
// (e.g. Scanner was still starting up). Without resetting the cached
// generation, the next poll would see roxagent report "unchanged"
// (report_generation didn't change) and skip resending, stranding the VM
// until mandatoryRefreshAfter (4h) instead of retrying on the next poll
// interval. It also verifies that a NACK is a no-op when it doesn't resolve
// to a currently-running, known VM (unrelated VM ID, malformed resource ID,
// or a VM that stopped running), and that an ACK never touches the cached
// generation.
func TestVMScraper_NACK(t *testing.T) {
	cases := map[string]struct {
		ackAction                  central.SensorACK_Action
		nackResourceID             string
		vmRunning                  bool
		pollResultAfterNack        *vsockclient.GetReportResult
		advanceAfterAck            time.Duration
		wantCalls                  int
		wantIfNewerThanOnRetryPoll uint32
		wantTotalSent              int
	}{
		"resets generation and resends after backoff when NACK matches the running VM": {
			ackAction:                  central.SensorACK_NACK,
			nackResourceID:             "vm-a-id:100",
			vmRunning:                  true,
			pollResultAfterNack:        makeReport(1),
			advanceAfterAck:            initialBackoff,
			wantCalls:                  1,
			wantIfNewerThanOnRetryPoll: 0,
			wantTotalSent:              2,
		},
		"is a no-op when NACK references an unrelated VM ID": {
			ackAction:                  central.SensorACK_NACK,
			nackResourceID:             "unknown-vm-id:999",
			vmRunning:                  true,
			pollResultAfterNack:        unchangedResult(),
			advanceAfterAck:            5 * time.Minute,
			wantCalls:                  1,
			wantIfNewerThanOnRetryPoll: 1,
			wantTotalSent:              1,
		},
		"is a no-op when the resource ID has no vsockCID suffix": {
			ackAction:                  central.SensorACK_NACK,
			nackResourceID:             "vm-a-id-with-no-colon",
			vmRunning:                  true,
			pollResultAfterNack:        unchangedResult(),
			advanceAfterAck:            5 * time.Minute,
			wantCalls:                  1,
			wantIfNewerThanOnRetryPoll: 1,
			wantTotalSent:              1,
		},
		"is a no-op when the NACKed VM is no longer running": {
			ackAction:           central.SensorACK_NACK,
			nackResourceID:      "vm-a-id:100",
			vmRunning:           false,
			pollResultAfterNack: unchangedResult(),
			advanceAfterAck:     5 * time.Minute,
			wantCalls:           0,
			wantTotalSent:       1,
		},
		"is a no-op for an ACK": {
			ackAction:                  central.SensorACK_ACK,
			nackResourceID:             "vm-a-id:100",
			vmRunning:                  true,
			pollResultAfterNack:        unchangedResult(),
			advanceAfterAck:            5 * time.Minute,
			wantCalls:                  1,
			wantIfNewerThanOnRetryPoll: 1,
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

			s, clock := newTestScraper(store, sender, dialer, client)
			s.pollOnce(context.Background())
			require.Len(t, sender.sent, 1)

			// Flip Running only after the first poll, so the VM is scraped normally
			// once before the ACK/NACK under test is delivered.
			vmA.Running = tc.vmRunning

			acksBefore := testutil.ToFloat64(metrics.IndexReportAcksReceived.WithLabelValues(tc.ackAction.String()))
			err := s.ProcessMessage(context.Background(), &central.MsgToSensor{
				Msg: &central.MsgToSensor_SensorAck{
					SensorAck: &central.SensorACK{
						MessageType: central.SensorACK_VM_INDEX_REPORT,
						Action:      tc.ackAction,
						ResourceId:  tc.nackResourceID,
					},
				},
			})
			require.NoError(t, err)
			assert.Equal(t, acksBefore+1, testutil.ToFloat64(metrics.IndexReportAcksReceived.WithLabelValues(tc.ackAction.String())))

			client.reset()
			client.resultQueue = []*vsockclient.GetReportResult{tc.pollResultAfterNack}
			clock.Advance(tc.advanceAfterAck)
			s.pollOnce(context.Background())

			require.Len(t, client.calls, tc.wantCalls)
			if tc.wantCalls > 0 {
				assert.Equal(t, tc.wantIfNewerThanOnRetryPoll, client.calls[0].ifNewerThan,
					"generation the scraper requests on the poll following the NACK")
			}
			assert.Len(t, sender.sent, tc.wantTotalSent, "total reports handed to the sender across both polls")
		})
	}
}

// gateSender blocks the Nth Send (1-based) until release is closed.
type gateSender struct {
	blockAt int
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
		<-g.release
	}
	return nil
}

// TestVMScraper_InFlightSendCanOverwriteNACKReset exercises a known race involving
// an unusually slow execution of `commitVMState` and a quick NACK from Central.
func TestVMScraper_InFlightSendCanOverwriteNACKReset(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		vmA := makeVM("ns1", "vm-a", 100)
		vmA.ID = "vm-a-id"
		store := &mockStore{vms: []*virtualmachine.Info{vmA}}
		client := &mockProtocolClient{resultQueue: []*vsockclient.GetReportResult{makeReport(1)}}
		s, clock := newTestScraper(store, &mockSender{}, &mockDialer{}, client)

		s.pollOnce(t.Context())
		require.Equal(t, uint32(1), cachedGeneration(t, s, "ns1/vm-a"))

		gate := &gateSender{
			blockAt: 1,
			release: make(chan struct{}),
		}
		s.sender = gate
		client.reset()
		client.resultQueue = []*vsockclient.GetReportResult{makeReport(2)}
		clock.Advance(s.interval)

		done := make(chan struct{})
		go func() {
			defer close(done)
			s.pollOnce(t.Context())
		}()

		// Block until the scrape goroutine is durably blocked inside Send,
		// i.e. the report has been handed off but not yet committed.
		synctest.Wait()

		require.NoError(t, s.ProcessMessage(t.Context(), &central.MsgToSensor{
			Msg: &central.MsgToSensor_SensorAck{
				SensorAck: &central.SensorACK{
					MessageType: central.SensorACK_VM_INDEX_REPORT,
					Action:      central.SensorACK_NACK,
					ResourceId:  "vm-a-id:100",
				},
			},
		}))
		assert.Equal(t, uint32(0), cachedGeneration(t, s, "ns1/vm-a"),
			"NACK applies immediately while the send it targets is still in flight")

		close(gate.release)
		<-done
		assert.Equal(t, uint32(2), cachedGeneration(t, s, "ns1/vm-a"),
			"the in-flight send's commit runs after the NACK reset and overwrites it unconditionally")
	})
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
			dialer: &mockDialer{},
			// Start order is hash-based; fail the first GetReport call so one VM
			// errors and the other still forwards.
			resultQueue: []*vsockclient.GetReportResult{nil, makeReport(1)},
			errQueue:    []error{errors.New("connection refused"), nil},
			wantCalls:   2,
			wantSent:    1,
		},
		"should still send for vm-b when vm-a dial fails": {
			dialer: &nameFailDialer{failName: "vm-a"},
			resultQueue: []*vsockclient.GetReportResult{
				makeReport(1),
			},
			wantCalls: 1,
			wantSent:  1,
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
			s, _ := newTestScraper(&mockStore{vms: vms}, sender, tc.dialer, client)
			if tc.perVMTimeout > 0 {
				s.perVMTimeout = tc.perVMTimeout
			}
			s.pollOnce(context.Background())

			assert.Len(t, client.calls, tc.wantCalls)
			assert.Len(t, sender.sent, tc.wantSent)
		})
	}
}

// nameFailDialer fails Dial for a single VM name so multi-VM tests need not
// depend on hash start order for errQueue alignment.
type nameFailDialer struct {
	failName string
}

func (d nameFailDialer) Dial(_ context.Context, _, name string, _ uint32, _ bool) (io.ReadWriteCloser, error) {
	if name == d.failName {
		return nil, errors.New("dial failed")
	}
	return nopCloser{}, nil
}

func TestVMScraper_StartRejectsSecondCall(t *testing.T) {
	s, _ := newTestScraper(&mockStore{}, &mockSender{}, &mockDialer{}, &mockProtocolClient{})
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
	s, _ := newTestScraper(&mockStore{vms: []*virtualmachine.Info{vm}}, &mockSender{}, &mockDialer{}, client)

	timeoutBefore := testutil.ToFloat64(metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusTimeout))
	readErrBefore := testutil.ToFloat64(metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusReadError))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, outcome := s.dialAndGetReport(ctx, vm, "ns1/vm-a", 1, 0, 0)

	assert.Equal(t, scrapeNonRetryable, outcome,
		"parent cancellation must not schedule a retry on the short tick")
	assert.Equal(t, timeoutBefore+1, testutil.ToFloat64(metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusTimeout)))
	assert.Equal(t, readErrBefore, testutil.ToFloat64(metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusReadError)),
		"timed-out read must not be counted as a protocol/read error")
}

// TestVMScraper_GetReportDeadlineExceededClassified covers a per-VM timeout
// (context.DeadlineExceeded), which is retried on the short tick, unlike the
// parent-cancellation case above.
func TestVMScraper_GetReportDeadlineExceededClassified(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)
	client := &mockProtocolClient{
		errQueue: []error{errors.New("i/o timeout")},
	}
	s, _ := newTestScraper(&mockStore{vms: []*virtualmachine.Info{vm}}, &mockSender{}, &mockDialer{}, client)

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()
	_, outcome := s.dialAndGetReport(ctx, vm, "ns1/vm-a", 1, 0, 0)

	assert.Equal(t, scrapeRetryable, outcome, "a per-VM deadline is retried on the short tick")
}

// TestVMScraper_GetReportBusyClassified verifies busy is counted separately
// from generic read/protocol errors and is retryable.
func TestVMScraper_GetReportBusyClassified(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)
	client := &mockProtocolClient{
		errQueue: []error{vsockclient.ErrBusy},
	}
	s, _ := newTestScraper(&mockStore{vms: []*virtualmachine.Info{vm}}, &mockSender{}, &mockDialer{}, client)

	busyBefore := testutil.ToFloat64(metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusBusy))
	readErrBefore := testutil.ToFloat64(metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusReadError))

	_, outcome := s.dialAndGetReport(context.Background(), vm, "ns1/vm-a", 1, 0, 0)

	assert.Equal(t, scrapeRetryable, outcome)
	assert.Equal(t, busyBefore+1, testutil.ToFloat64(metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusBusy)))
	assert.Equal(t, readErrBefore, testutil.ToFloat64(metrics.PullRequestsTotal.WithLabelValues(metrics.PullStatusReadError)),
		"a busy response must not be counted as a generic protocol/read error")
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

	s, clock := newTestScraper(store, sender, dialer, client)
	s.pollOnce(context.Background())
	assert.Len(t, s.vmState, 2)
	assert.True(t, hasScheduleSlot(t, s, "ns1/vm-a"))

	// Remove vm-a from running set
	store.vms = []*virtualmachine.Info{makeVM("ns2", "vm-b", 200)}
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{makeReport(2)}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())

	assert.Len(t, s.vmState, 1, "stale vm-a state should be pruned")
	assert.False(t, hasScheduleSlot(t, s, "ns1/vm-a"), "vm-a should no longer be scheduled")
	assert.True(t, hasScheduleSlot(t, s, "ns2/vm-b"))
}

// hasScheduleSlot reports whether key has a vmState slot (reconcile membership).
func hasScheduleSlot(t *testing.T, s *VMScraper, key string) bool {
	t.Helper()
	return concurrency.WithLock1(&s.mu, func() bool {
		_, ok := s.vmState[key]
		return ok
	})
}

// cachedGeneration reads a VM's cached generation under s.mu, so the read is
// race-safe regardless of what locking handleNACK or commitVMState use internally.
func cachedGeneration(t *testing.T, s *VMScraper, key string) uint32 {
	t.Helper()
	return concurrency.WithLock1(&s.mu, func() uint32 {
		st, ok := s.vmState[key]
		require.True(t, ok, "no cached state for %q", key)
		return st.lastGeneration
	})
}

func newTestScraper(store RunningVMStore, sender IndexReportSender, dialer VMDialer, client ProtocolClient) (*VMScraper, *testClock) {
	clock := newTestClock()
	interval := 5 * time.Minute
	return &VMScraper{
		store:    store,
		sender:   sender,
		dialer:   dialer,
		client:   client,
		interval: interval,
		// Match catch-up window so default urgent due sets drain in one tick
		// under concurrency (production uses initialBackoff; pacing tests override).
		tickInterval:          catchUpWindow(interval),
		initialBackoff:        initialBackoff,
		reconcileEvery:        reconcilePeriod(interval),
		perVMTimeout:          10 * time.Second,
		mandatoryRefreshAfter: 4 * time.Hour,
		concurrency:           20,
		spreadFraction:        2.0 / 3,
		// Half of the 16MiB default pull response-size ceiling — same
		// derivation New() uses from env.VirtualMachinesPullMaxResponseSizeKB.
		warnMaxBytes: 8 << 20,
		vmState:      make(map[string]*vmState),
		inFlight:     set.NewStringSet(),
		now:          clock.Now,
		// Zero offset keeps default schedule assertions deterministic - no randomization of the scrape schedule
		randFloat64: func() float64 { return 0 },
	}, clock
}

// pollOnce forces a reconcile and scrapes every due slot.
func (s *VMScraper) pollOnce(ctx context.Context) {
	s.tick(ctx, true)
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

	s, _ := newTestScraper(store, sender, dialer, client)
	s.concurrency = concurrency
	// Catch-up budget is ceil(nUrgent × tick / catchUp); set tick >= catchUp
	// so one tick can start the whole fleet and exercise errgroup concurrency.
	s.interval = 30 * time.Second
	s.tickInterval = 10 * time.Second // catchUp = interval/3 = 10s → budget = numVMs
	s.reconcileEvery = reconcilePeriod(s.interval)
	// This test measures real dial overlap via delayDialer's timers, so it
	// needs the wall clock rather than the fake test clock.
	s.now = time.Now

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

func TestVMScraper_RetryableFailureSchedulesBackoff(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{makeVM("ns1", "vm-a", 100)}}
	client := &mockProtocolClient{
		errQueue: []error{vsockclient.ErrNotReady},
	}
	s, clock := newTestScraper(store, &mockSender{}, &mockDialer{}, client)

	s.pollOnce(context.Background())
	require.Len(t, client.calls, 1)
	assert.Equal(t, initialBackoff, cachedBackoff(t, s, "ns1/vm-a"))

	client.reset()
	client.errQueue = nil
	client.resultQueue = []*vsockclient.GetReportResult{makeReport(1)}
	s.pollOnce(context.Background())
	assert.Len(t, client.calls, 0, "should skip while backoff has not elapsed")

	clock.Advance(initialBackoff)
	s.pollOnce(context.Background())
	assert.Len(t, client.calls, 1)
	assert.Zero(t, cachedBackoff(t, s, "ns1/vm-a"), "success resets the backoff")

	// A second consecutive retryable failure doubles the backoff.
	client.reset()
	client.errQueue = []error{vsockclient.ErrNotReady}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())
	require.Len(t, client.calls, 1)
	assert.Equal(t, initialBackoff, cachedBackoff(t, s, "ns1/vm-a"))

	client.reset()
	client.errQueue = []error{vsockclient.ErrNotReady}
	clock.Advance(initialBackoff)
	s.pollOnce(context.Background())
	assert.Equal(t, 2*initialBackoff, cachedBackoff(t, s, "ns1/vm-a"), "a second consecutive failure doubles the backoff")
}

// TestVMScraper_SchedulesByOutcome pins send-error short backoff vs
// permanent-failure poll cadence (no backoff growth).
func TestVMScraper_SchedulesByOutcome(t *testing.T) {
	const key = "ns1/vm-a"
	cases := map[string]struct {
		client      *mockProtocolClient
		sender      *mockSender
		wantBackoff time.Duration
		wantGap     time.Duration
	}{
		"send failure should retry using backoff": {
			client:      &mockProtocolClient{resultQueue: []*vsockclient.GetReportResult{makeReport(1)}},
			sender:      &mockSender{err: errors.New("central unavailable")},
			wantBackoff: initialBackoff,
			wantGap:     initialBackoff,
		},
		"ErrInternal should retry using backoff": {
			client:      &mockProtocolClient{errQueue: []error{vsockclient.ErrInternal}},
			sender:      &mockSender{},
			wantBackoff: initialBackoff,
			wantGap:     initialBackoff,
		},
		"io.EOF should retry using backoff": {
			client:      &mockProtocolClient{errQueue: []error{io.EOF}},
			sender:      &mockSender{},
			wantBackoff: initialBackoff,
			wantGap:     initialBackoff,
		},
		"io.ErrUnexpectedEOF should retry using backoff": {
			client:      &mockProtocolClient{errQueue: []error{io.ErrUnexpectedEOF}},
			sender:      &mockSender{},
			wantBackoff: initialBackoff,
			wantGap:     initialBackoff,
		},
		"unhandled protocol error should retry using backoff": {
			client:      &mockProtocolClient{errQueue: []error{errors.New("bogus frame")}},
			sender:      &mockSender{},
			wantBackoff: initialBackoff,
			wantGap:     initialBackoff,
		},
		"ErrUnknownMethod should not retry using backoff": {
			client:      &mockProtocolClient{errQueue: []error{vsockclient.ErrUnknownMethod}},
			sender:      &mockSender{},
			wantBackoff: 0,
			wantGap:     5 * time.Minute, // matches newTestScraper interval
		},
		"invalid report should not retry using backoff": {
			client: &mockProtocolClient{resultQueue: []*vsockclient.GetReportResult{{
				Meta: &pb.ResponseMeta{ReportGeneration: 1},
			}}},
			sender:      &mockSender{},
			wantBackoff: 0,
			wantGap:     5 * time.Minute,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s, clock := newTestScraper(
				&mockStore{vms: []*virtualmachine.Info{makeVM("ns1", "vm-a", 100)}},
				tc.sender, &mockDialer{}, tc.client,
			)
			start := clock.Now()
			s.pollOnce(context.Background())

			assert.Equal(t, tc.wantBackoff, cachedBackoff(t, s, key))
			assert.Equal(t, tc.wantGap, cachedNextAttemptAt(t, s, key).Sub(start))
		})
	}
}

// TestVMScraper_NonForcedTickSkipsReconcileWhenNotDue verifies that the
// production ticker's non-forced tick only reconciles (calling ListRunning)
// once lastReconcile is at least reconcileEvery old, keeping ListRunning off
// the hot per-tick path.
func TestVMScraper_NonForcedTickSkipsReconcileWhenNotDue(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{makeVM("ns1", "vm-a", 100)}}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport(1), makeReport(1)},
	}
	s, clock := newTestScraper(store, &mockSender{}, &mockDialer{}, client)

	s.pollOnce(context.Background())
	require.Equal(t, 1, store.listRunningCalls, "pollOnce forces exactly one reconcile")

	clock.Advance(s.reconcileEvery / 2)
	s.tick(context.Background(), false)
	assert.Equal(t, 1, store.listRunningCalls, "reconcile is not yet due")

	clock.Advance(s.reconcileEvery / 2)
	s.tick(context.Background(), false)
	assert.Equal(t, 2, store.listRunningCalls, "reconcile becomes due once reconcileEvery has elapsed")
}

func cachedBackoff(t *testing.T, s *VMScraper, key string) time.Duration {
	t.Helper()
	return concurrency.WithLock1(&s.mu, func() time.Duration {
		st, ok := s.vmState[key]
		require.True(t, ok, "no cached state for %q", key)
		return st.backoff
	})
}

func cachedNextAttemptAt(t *testing.T, s *VMScraper, key string) time.Time {
	t.Helper()
	return concurrency.WithLock1(&s.mu, func() time.Time {
		st, ok := s.vmState[key]
		require.True(t, ok, "no cached state for %q", key)
		return st.nextAttemptAt
	})
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
