package vmscraper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
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
	err error
}

func (m *mockDialer) Dial(_ context.Context, _, _ string, _ uint32, _ bool) (io.ReadWriteCloser, error) {
	if m.err != nil {
		return nil, m.err
	}
	return nopCloser{}, nil
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

func (m *mockProtocolClient) GetReport(_ io.ReadWriteCloser, ifNewerThan uint32, knownEpoch uint32) (*vsockclient.GetReportResult, error) {
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

	// Simulate 4h+1s elapsed
	s.now = func() time.Time { return time.Now().Add(mandatoryRefreshAfter + time.Second) }

	// First call returns unchanged, second call (forced refresh gen=0) returns full report
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResult(), makeReport(1)}
	s.pollOnce(context.Background())

	require.Len(t, client.calls, 2)
	assert.Equal(t, uint32(1), client.calls[0].ifNewerThan, "first call uses last generation")
	assert.Equal(t, uint32(0), client.calls[1].ifNewerThan, "second call forces full report")
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

func TestVMScraper_HandlesDialError(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
		makeVM("ns2", "vm-b", 200),
	}}
	sender := &mockSender{}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{nil, makeReport(1)},
		errQueue:    []error{errors.New("connection refused"), nil},
	}

	s := newTestScraper(store, sender, dialer, client)
	s.pollOnce(context.Background())

	assert.Len(t, sender.sent, 1, "should still send for vm-b despite vm-a protocol error")
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
		store:       store,
		sender:      sender,
		dialer:      dialer,
		client:      client,
		interval:    5 * time.Minute,
		concurrency: 1,
		vmState:     make(map[string]*vmState),
		activeVMs:   set.NewStringSet(),
		now:         time.Now,
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

func (c *safeProtocolClient) GetReport(_ io.ReadWriteCloser, _ uint32, _ uint32) (*vsockclient.GetReportResult, error) {
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
		store:       store,
		sender:      sender,
		dialer:      dialer,
		client:      client,
		interval:    5 * time.Minute,
		concurrency: concurrency,
		vmState:     make(map[string]*vmState),
		activeVMs:   set.NewStringSet(),
		now:         time.Now,
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
