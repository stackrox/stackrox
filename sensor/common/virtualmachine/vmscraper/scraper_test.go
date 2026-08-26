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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	pkgVM "github.com/stackrox/rox/pkg/virtualmachine"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// --- Mocks ---

// mockStore is used by concurrent scrape workers (persistAgentFacts calls
// AddOrUpdate), so mu guards vms and listRunningCalls.
type mockStore struct {
	mu               sync.Mutex
	vms              []*virtualmachine.Info
	listRunningCalls int
}

// ListRunning mirrors the production store's contract of only returning VMs
// with Running set, so tests exercise reconcile pruning rather than the
// scrapeKey nil-VM guard when a VM stops running.
func (m *mockStore) ListRunning() []*virtualmachine.Info {
	return concurrency.WithLock1(&m.mu, func() []*virtualmachine.Info {
		m.listRunningCalls++
		var out []*virtualmachine.Info
		for _, vm := range m.vms {
			if vm.Running {
				out = append(out, vm)
			}
		}
		return out
	})
}

func (m *mockStore) Get(id virtualmachine.VMID) *virtualmachine.Info {
	return concurrency.WithLock1(&m.mu, func() *virtualmachine.Info {
		for _, vm := range m.vms {
			if vm.ID == id {
				return vm.Copy()
			}
		}
		return nil
	})
}

func (m *mockStore) AddOrUpdate(vm *virtualmachine.Info) *virtualmachine.Info {
	if vm == nil {
		return nil
	}
	return concurrency.WithLock1(&m.mu, func() *virtualmachine.Info {
		for i, existing := range m.vms {
			if existing.ID != vm.ID {
				continue
			}
			if vm.AgentFacts == nil {
				vm.AgentFacts = existing.AgentFacts
			}
			m.vms[i] = vm
			return vm
		}
		m.vms = append(m.vms, vm)
		return vm
	})
}

type mockDialer struct {
	err      error
	errQueue []error
	callIdx  atomic.Int32
}

func (m *mockDialer) Dial(_ context.Context, _, _ string, _ uint32, _ bool) (io.ReadWriteCloser, error) {
	idx := int(m.callIdx.Add(1) - 1)
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

// fakeCloseCoder is a minimal closeCoder for testing without a real dialer.
type fakeCloseCoder struct {
	code   int
	reason string
}

func (e *fakeCloseCoder) Error() string            { return "connection closed" }
func (e *fakeCloseCoder) CloseCode() (int, string) { return e.code, e.reason }

// singleConnDialer rejects a Dial while a previously dialed stream is still
// open, mirroring roxagent's real maxConcurrentConns=1 semaphore, so a
// caller that dials again before closing its current connection sees the
// same failure it would against the real agent.
type singleConnDialer struct {
	mu   sync.Mutex
	open bool
}

func (d *singleConnDialer) Dial(_ context.Context, _, _ string, _ uint32, _ bool) (io.ReadWriteCloser, error) {
	return concurrency.WithLock2(&d.mu, func() (io.ReadWriteCloser, error) {
		if d.open {
			return nil, errors.New("agent is already serving another request; retry after a backoff")
		}
		d.open = true
		return &trackedStream{dialer: d}, nil
	})
}

type trackedStream struct {
	dialer *singleConnDialer
}

func (t *trackedStream) Read([]byte) (int, error)  { return 0, io.EOF }
func (t *trackedStream) Write([]byte) (int, error) { return 0, nil }
func (t *trackedStream) Close() error {
	concurrency.WithLock(&t.dialer.mu, func() { t.dialer.open = false })
	return nil
}

type mockProtocolClient struct {
	mu          sync.Mutex
	resultQueue []*vsockclient.GetReportResult
	errQueue    []error
	calls       []protocolCall
	callIdx     int

	getReportDelay  time.Duration
	syncDelay       time.Duration
	syncBlocksOnCtx bool
	syncCalls       [][]byte
	syncUpdated     bool
	syncMeta        *pb.ResponseMeta
	syncErr         error
}

type protocolCall struct {
	lastKnownToken string
}

func (m *mockProtocolClient) GetReport(_ context.Context, _ io.ReadWriteCloser, lastKnownToken string) (*vsockclient.GetReportResult, error) {
	if m.getReportDelay > 0 {
		time.Sleep(m.getReportDelay)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, protocolCall{lastKnownToken: lastKnownToken})
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

func (m *mockProtocolClient) SyncRepoCPEMapping(ctx context.Context, _ io.ReadWriteCloser, mapping []byte) (bool, *pb.ResponseMeta, error) {
	if m.syncBlocksOnCtx {
		m.syncCalls = append(m.syncCalls, mapping)
		<-ctx.Done()
		return false, nil, ctx.Err()
	}
	if m.syncDelay > 0 {
		time.Sleep(m.syncDelay)
	}
	m.syncCalls = append(m.syncCalls, mapping)
	return m.syncUpdated, m.syncMeta, m.syncErr
}

func (m *mockProtocolClient) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func makeReport(token string) *vsockclient.GetReportResult {
	return &vsockclient.GetReportResult{
		IndexReport: &v4.IndexReport{
			State: "IndexFinished",
		},
		Meta: &pb.ResponseMeta{
			ReportToken: token,
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
		Meta:      &pb.ResponseMeta{ReportToken: "1"},
	}
}

func metaWithMapping(hash string, path pb.RepoCPEMappingUpdatePath) *pb.ResponseMeta {
	return &pb.ResponseMeta{
		RepoCpeMappingHash:       new(hash),
		RepoCpeMappingUpdatePath: path.Enum(),
	}
}

func unchangedResultWithFacts(facts map[string]string) *vsockclient.GetReportResult {
	return &vsockclient.GetReportResult{
		Unchanged: true,
		Meta: &pb.ResponseMeta{
			ReportToken: "1",
			Facts:       facts,
		},
	}
}

// --- Tests ---

func TestVMScraper_PollsRunningVMs(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
		makeVM("ns2", "vm-b", 200),
	}}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport("1"), makeReport("1")},
		errQueue:    []error{nil, nil},
	}

	s, _ := newTestScraper(t, store, dialer, client)
	discoveredBefore := testutil.ToFloat64(metrics.VMDiscoveredData.WithLabelValues("RHEL", "ACTIVE", "AVAILABLE"))
	s.pollOnce(context.Background())

	assert.Equal(t, 2, forwardedCount(s))
	assert.Len(t, client.calls, 2)
	assert.Equal(t, discoveredBefore+2, testutil.ToFloat64(metrics.VMDiscoveredData.WithLabelValues("RHEL", "ACTIVE", "AVAILABLE")))
	expectedFacts := virtualmachine.AgentFactsFromResponseFacts(map[string]string{
		"detected_os":         "RHEL",
		"activation_status":   "ACTIVE",
		"dnf_metadata_status": "AVAILABLE",
	})
	assert.Equal(t, expectedFacts, store.Get(virtualmachine.VMID("ns1/vm-a")).AgentFacts)
	assert.Equal(t, expectedFacts, store.Get(virtualmachine.VMID("ns2/vm-b")).AgentFacts)
}

func TestPersistAgentFactsDoesNotAliasStorePointer(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)
	store := &mockStore{vms: []*virtualmachine.Info{vm.Copy()}}
	s := &VMScraper{store: store}

	s.persistAgentFacts(vm, map[string]string{
		"detected_os":         "RHEL",
		"activation_status":   "ACTIVE",
		"dnf_metadata_status": "AVAILABLE",
	})
	vm.GuestOS = "mutated-after-persist"

	stored := store.Get(vm.ID)
	require.NotNil(t, stored)
	assert.NotEqual(t, "mutated-after-persist", stored.GuestOS)
	assert.Equal(t, virtualmachine.AgentFactsFromResponseFacts(map[string]string{
		"detected_os":         "RHEL",
		"activation_status":   "ACTIVE",
		"dnf_metadata_status": "AVAILABLE",
	}), stored.AgentFacts)
}

// TestVMScraper_SkipsWhenCentralLacksCapability covers a store already
// populated (for example after reconnecting to an older Central) so the
// scraper must not dial or forward.
func TestVMScraper_SkipsWhenCentralLacksCapability(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
	}

	s, _ := newTestScraper(t, store, dialer, client)
	centralcaps.Set(nil)

	s.pollOnce(context.Background())

	assert.Equal(t, 0, store.listRunningCalls, "should not reconcile when Central cannot consume reports")
	assert.Zero(t, dialer.callIdx.Load())
	assert.Zero(t, forwardedCount(s))
	assert.Empty(t, client.calls)
}

// TestVMScraper_LogsSkipOnceWhileCapabilityMissing covers a missing-capability
// stretch that lasts more than one tick, then a later drop after the
// capability returns, so the skip log fires once per stretch not once per tick.
func TestVMScraper_LogsSkipOnceWhileCapabilityMissing(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	orig := log
	log = &logging.LoggerImpl{InnerLogger: zap.New(core).Sugar()}
	t.Cleanup(func() { log = orig })

	s, _ := newTestScraper(t, &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}, &mockDialer{}, &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
	})
	centralcaps.Set(nil)

	s.pollOnce(t.Context())
	s.pollOnce(t.Context())
	assert.Equal(t, 1, logs.FilterMessageSnippet("skipping pull").Len())

	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported})
	s.pollOnce(t.Context())

	centralcaps.Set(nil)
	s.pollOnce(t.Context())
	assert.Equal(t, 2, logs.FilterMessageSnippet("skipping pull").Len())
}

func TestVMScraper_SkipsUnchangedToken(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
	}

	s, clock := newTestScraper(t, store, dialer, client)

	s.pollOnce(context.Background())
	require.Equal(t, 1, forwardedCount(s))

	// Second poll returns unchanged
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResult()}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())
	assert.Zero(t, forwardedCount(s), "should not forward unchanged report")
}

func TestVMScraper_ForwardsChangedAgentFactsOnUnchangedReport(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
	}

	s, clock := newTestScraper(t, store, dialer, client)
	s.pollOnce(context.Background())
	require.Equal(t, 1, forwardedCount(s))

	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResultWithFacts(map[string]string{
		"detected_os":         "RHEL",
		"activation_status":   "INACTIVE",
		"dnf_metadata_status": "UNAVAILABLE",
	})}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())

	expected := virtualmachine.AgentFactsFromResponseFacts(map[string]string{
		"detected_os":         "RHEL",
		"activation_status":   "INACTIVE",
		"dnf_metadata_status": "UNAVAILABLE",
	})
	updates := drainVMUpdates(s)
	require.Len(t, updates, 1, "unchanged report should not be forwarded")
	assert.Equal(t, expected[pkgVM.ActivationStatusKey], updates[0].GetFacts()[pkgVM.ActivationStatusKey])
	assert.Equal(t, expected, store.Get(virtualmachine.VMID("ns1/vm-a")).AgentFacts)
}

func TestVMScraper_DoesNotResendUnchangedAgentFacts(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
	}

	s, clock := newTestScraper(t, store, dialer, client)
	s.pollOnce(context.Background())
	require.Equal(t, 1, forwardedCount(s))

	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResultWithFacts(map[string]string{
		"detected_os":         "RHEL",
		"activation_status":   "ACTIVE",
		"dnf_metadata_status": "AVAILABLE",
	})}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())
	assert.Empty(t, drainToCentral(s), "should not emit a VM update when agent facts are unchanged")
}

func TestVMScraper_RetriesAgentFactsAfterSendFailure(t *testing.T) {
	vmID := virtualmachine.VMID("ns1/vm-a")
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
	}

	s, clock := newTestScraper(t, store, dialer, client)
	s.pollOnce(context.Background())
	require.Equal(t, 1, forwardedCount(s))
	initialFacts := virtualmachine.AgentFactsFromResponseFacts(map[string]string{
		"detected_os":         "RHEL",
		"activation_status":   "ACTIVE",
		"dnf_metadata_status": "AVAILABLE",
	})
	assert.Equal(t, initialFacts, store.Get(vmID).AgentFacts)

	changedFacts := map[string]string{
		"detected_os":         "RHEL",
		"activation_status":   "INACTIVE",
		"dnf_metadata_status": "UNAVAILABLE",
	}
	s.centralReady.Reset()
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResultWithFacts(changedFacts)}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())

	assert.Empty(t, drainToCentral(s), "failed metadata send should not count as forwarded")
	assert.Equal(t, initialFacts, store.Get(vmID).AgentFacts)

	s.centralReady.Signal()
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResultWithFacts(changedFacts)}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())

	expected := virtualmachine.AgentFactsFromResponseFacts(changedFacts)
	updates := drainVMUpdates(s)
	require.Len(t, updates, 1, "retry should be metadata-only")
	assert.Equal(t, expected[pkgVM.ActivationStatusKey], updates[0].GetFacts()[pkgVM.ActivationStatusKey])
	assert.Equal(t, expected, store.Get(vmID).AgentFacts)
}

func TestVMScraper_ClearsAgentFactsWhenResponseValuesUnspecified(t *testing.T) {
	vmID := virtualmachine.VMID("ns1/vm-a")
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
	}

	s, clock := newTestScraper(t, store, dialer, client)
	s.pollOnce(context.Background())
	require.Equal(t, 1, forwardedCount(s))
	require.NotEmpty(t, store.Get(vmID).AgentFacts)

	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResultWithFacts(map[string]string{
		"detected_os":         pb.DetectedOS_UNKNOWN.String(),
		"activation_status":   pb.ActivationStatus_ACTIVATION_UNSPECIFIED.String(),
		"dnf_metadata_status": pb.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED.String(),
	})}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())

	updates := drainVMUpdates(s)
	require.Len(t, updates, 1, "clearing facts should be metadata-only")
	assert.Empty(t, store.Get(vmID).AgentFacts)
}

func TestVMScraper_RemainsScheduledAcrossUnchangedPolls(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
	}

	s, clock := newTestScraper(t, store, dialer, client)

	s.pollOnce(context.Background())
	require.True(t, hasScheduleSlot(t, s, "ns1/vm-a"))
	require.Equal(t, 1, forwardedCount(s))

	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResult()}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())

	assert.Zero(t, forwardedCount(s), "should not forward unchanged report")
	assert.True(t, hasScheduleSlot(t, s, "ns1/vm-a"))
}

func TestVMScraper_ForwardsAfter4Hours(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
	}

	s, clock := newTestScraper(t, store, dialer, client)

	s.pollOnce(context.Background())
	require.Equal(t, 1, forwardedCount(s))

	// Past both the success schedule and the mandatory refresh window.
	clock.Advance(s.mandatoryRefreshAfter + time.Second)

	// The mandatory refresh is known before dialing, so a single call
	// requesting the full report (empty last_known_token) is enough.
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{makeReport("1")}
	s.pollOnce(context.Background())

	require.Len(t, client.calls, 1, "mandatory refresh should resolve in a single round trip")
	assert.Empty(t, client.calls[0].lastKnownToken, "mandatory refresh forces the full report on the only call")
	assert.Equal(t, 1, forwardedCount(s), "should forward after 4h even if unchanged")
}

// TestVMScraper_SendsLastKnownTokenOnRequest verifies Sensor sends its
// last-cached token on every request so a matching scan reports unchanged.
func TestVMScraper_SendsLastKnownTokenOnRequest(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport("tok-100")},
	}

	s, clock := newTestScraper(t, store, dialer, client)
	s.pollOnce(context.Background())

	require.Len(t, client.calls, 1)
	assert.Empty(t, client.calls[0].lastKnownToken, "first-ever request for a VM has no cached token")

	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{unchangedResult()}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())

	require.Len(t, client.calls, 1, "matching token should resolve in a single round trip")
	assert.Equal(t, "tok-100", client.calls[0].lastKnownToken, "subsequent requests send the cached token")
}

func TestVMScraper_ForwardsOnTokenChange(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns1", "vm-a", 100),
	}}
	dialer := &mockDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
	}

	s, clock := newTestScraper(t, store, dialer, client)
	s.pollOnce(context.Background())
	require.Equal(t, 1, forwardedCount(s))

	// New token
	client.reset()
	client.resultQueue = []*vsockclient.GetReportResult{makeReport("2")}
	clock.Advance(s.interval)
	s.pollOnce(context.Background())
	assert.Equal(t, 1, forwardedCount(s), "should forward on token change")
}

// TestVMScraper_NACK reproduces a scenario where Central NACKs a report
// (e.g. Scanner was still starting up). Without resetting the cached
// token, the next poll would see roxagent report "unchanged" and skip
// resending, stranding the VM until mandatoryRefreshAfter (4h) instead of
// retrying on the next poll interval. It also verifies that a NACK is a
// no-op when it doesn't resolve to a currently-running, known VM (unrelated
// VM ID, malformed resource ID, or a VM that stopped running), and that an
// ACK never touches the cached token.
func TestVMScraper_NACK(t *testing.T) {
	cases := map[string]struct {
		ackAction                     central.SensorACK_Action
		nackResourceID                string
		vmRunning                     bool
		pollResultAfterNack           *vsockclient.GetReportResult
		advanceAfterAck               time.Duration
		wantCalls                     int
		wantLastKnownTokenOnRetryPoll string
		wantTotalSent                 int
	}{
		"resets token and resends after backoff when NACK matches the running VM": {
			ackAction:                     central.SensorACK_NACK,
			nackResourceID:                "vm-a-id:100",
			vmRunning:                     true,
			pollResultAfterNack:           makeReport("1"),
			advanceAfterAck:               initialBackoff,
			wantCalls:                     1,
			wantLastKnownTokenOnRetryPoll: "",
			wantTotalSent:                 2,
		},
		"is a no-op when NACK references an unrelated VM ID": {
			ackAction:                     central.SensorACK_NACK,
			nackResourceID:                "unknown-vm-id:999",
			vmRunning:                     true,
			pollResultAfterNack:           unchangedResult(),
			advanceAfterAck:               5 * time.Minute,
			wantCalls:                     1,
			wantLastKnownTokenOnRetryPoll: "1",
			wantTotalSent:                 1,
		},
		"is a no-op when the resource ID has no vsockCID suffix": {
			ackAction:                     central.SensorACK_NACK,
			nackResourceID:                "vm-a-id-with-no-colon",
			vmRunning:                     true,
			pollResultAfterNack:           unchangedResult(),
			advanceAfterAck:               5 * time.Minute,
			wantCalls:                     1,
			wantLastKnownTokenOnRetryPoll: "1",
			wantTotalSent:                 1,
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
			ackAction:                     central.SensorACK_ACK,
			nackResourceID:                "vm-a-id:100",
			vmRunning:                     true,
			pollResultAfterNack:           unchangedResult(),
			advanceAfterAck:               5 * time.Minute,
			wantCalls:                     1,
			wantLastKnownTokenOnRetryPoll: "1",
			wantTotalSent:                 1,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vmA := makeVM("ns1", "vm-a", 100)
			vmA.ID = "vm-a-id"
			store := &mockStore{vms: []*virtualmachine.Info{vmA}}
			dialer := &mockDialer{}
			client := &mockProtocolClient{
				resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
			}

			s, clock := newTestScraper(t, store, dialer, client)
			s.pollOnce(context.Background())
			require.Equal(t, 1, forwardedCount(s))

			// Flip Running only after the first poll, so the VM is scraped normally
			// once before the ACK/NACK under test is delivered. persistAgentFacts
			// stores a copy, so mutate the store's current object rather than vmA.
			concurrency.WithLock(&store.mu, func() {
				store.vms[0].Running = tc.vmRunning
			})

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
				assert.Equal(t, tc.wantLastKnownTokenOnRetryPoll, client.calls[0].lastKnownToken,
					"token the scraper requests on the poll following the NACK")
			}
			assert.Equal(t, tc.wantTotalSent-1, forwardedCount(s), "additional reports forwarded after the ACK/NACK poll")
		})
	}
}

// TestVMScraper_InFlightSendCanOverwriteNACKReset exercises a known race involving
// an unusually slow execution of `commitVMState` and a quick NACK from Central.
func TestVMScraper_InFlightSendCanOverwriteNACKReset(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		vmA := makeVM("ns1", "vm-a", 100)
		vmA.ID = "vm-a-id"
		store := &mockStore{vms: []*virtualmachine.Info{vmA}}
		client := &mockProtocolClient{resultQueue: []*vsockclient.GetReportResult{makeReport("1")}}
		s, clock := newTestScraper(t, store, &mockDialer{}, client)

		s.pollOnce(t.Context())
		require.Equal(t, "1", cachedToken(t, s, "ns1/vm-a"))
		require.Equal(t, 1, forwardedCount(s))

		s.toCentral = make(chan *message.ExpiringMessage)
		client.reset()
		client.resultQueue = []*vsockclient.GetReportResult{makeReport("2")}
		clock.Advance(s.interval)

		done := make(chan struct{})
		go func() {
			defer close(done)
			s.pollOnce(t.Context())
		}()

		// Block until the scrape goroutine is durably blocked in forwardReport,
		// i.e. the report is on toCentral but not yet committed.
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
		assert.Empty(t, cachedToken(t, s, "ns1/vm-a"),
			"NACK applies immediately while the send it targets is still in flight")

		<-s.toCentral
		<-done
		assert.Equal(t, "2", cachedToken(t, s, "ns1/vm-a"),
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
			dialer:      &mockDialer{},
			resultQueue: []*vsockclient.GetReportResult{nil, makeReport("1")},
			errQueue:    []error{errors.New("connection refused"), nil},
			wantCalls:   2,
			wantSent:    1,
		},
		"should still send for vm-b when vm-a dial fails": {
			dialer: &mockDialer{
				errQueue: []error{errors.New("dial failed"), nil},
			},
			resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
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
			client := &mockProtocolClient{
				resultQueue: tc.resultQueue,
				errQueue:    tc.errQueue,
			}
			s, _ := newTestScraper(t, &mockStore{vms: vms}, tc.dialer, client)
			if tc.perVMTimeout > 0 {
				s.perVMTimeout = tc.perVMTimeout
			}
			s.pollOnce(context.Background())

			assert.Len(t, client.calls, tc.wantCalls)
			assert.Equal(t, tc.wantSent, forwardedCount(s))
		})
	}
}

func TestVMScraper_StartRejectsSecondCall(t *testing.T) {
	s, _ := newTestScraper(t, &mockStore{}, &mockDialer{}, &mockProtocolClient{})
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
	s, _ := newTestScraper(t, &mockStore{vms: []*virtualmachine.Info{vm}}, &mockDialer{}, client)

	timeoutBefore := testutil.ToFloat64(metrics.PullTransportTotal.WithLabelValues(metrics.PullTransportTimeout))
	readErrBefore := testutil.ToFloat64(metrics.PullTransportTotal.WithLabelValues(metrics.PullTransportReadError))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, outcome := s.dialAndGetReport(ctx, vm, "ns1/vm-a", 1, "")

	assert.Equal(t, scrapeNonRetryable, outcome,
		"parent cancellation must not schedule a retry on the short tick")
	assert.Equal(t, timeoutBefore+1, testutil.ToFloat64(metrics.PullTransportTotal.WithLabelValues(metrics.PullTransportTimeout)))
	assert.Equal(t, readErrBefore, testutil.ToFloat64(metrics.PullTransportTotal.WithLabelValues(metrics.PullTransportReadError)),
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
	s, _ := newTestScraper(t, &mockStore{vms: []*virtualmachine.Info{vm}}, &mockDialer{}, client)

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()
	_, outcome := s.dialAndGetReport(ctx, vm, "ns1/vm-a", 1, "")

	assert.Equal(t, scrapeRetryable, outcome, "a per-VM deadline is retried on the short tick")
}

// TestVMScraper_GetReportBusyClassified verifies busy is counted separately
// from generic read/protocol errors and is retryable.
func TestVMScraper_GetReportBusyClassified(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)
	client := &mockProtocolClient{
		errQueue: []error{vsockclient.ErrBusy},
	}
	s, _ := newTestScraper(t, &mockStore{vms: []*virtualmachine.Info{vm}}, &mockDialer{}, client)

	busyBefore := testutil.ToFloat64(metrics.PullGetReportTotal.WithLabelValues(metrics.PullGetReportBusy))
	readErrBefore := testutil.ToFloat64(metrics.PullTransportTotal.WithLabelValues(metrics.PullTransportReadError))

	_, outcome := s.dialAndGetReport(context.Background(), vm, "ns1/vm-a", 1, "")

	assert.Equal(t, scrapeRetryable, outcome)
	assert.Equal(t, busyBefore+1, testutil.ToFloat64(metrics.PullGetReportTotal.WithLabelValues(metrics.PullGetReportBusy)))
	assert.Equal(t, readErrBefore, testutil.ToFloat64(metrics.PullTransportTotal.WithLabelValues(metrics.PullTransportReadError)),
		"a busy response must not be counted as a generic protocol/read error")
}

func TestVMScraper_ReconcileVMState(t *testing.T) {
	t.Run("should prune keys no longer in the running set", func(t *testing.T) {
		store := &mockStore{vms: []*virtualmachine.Info{
			makeVM("ns1", "vm-a", 100),
			makeVM("ns2", "vm-b", 200),
		}}
		dialer := &mockDialer{}
		client := &mockProtocolClient{
			resultQueue: []*vsockclient.GetReportResult{makeReport("1"), makeReport("1")},
		}

		s, clock := newTestScraper(t, store, dialer, client)
		s.pollOnce(context.Background())
		assert.Len(t, s.vmState, 2)
		assert.True(t, hasScheduleSlot(t, s, "ns1/vm-a"))

		store.vms = []*virtualmachine.Info{makeVM("ns2", "vm-b", 200)}
		client.reset()
		client.resultQueue = []*vsockclient.GetReportResult{makeReport("2")}
		clock.Advance(s.interval)
		s.pollOnce(context.Background())

		assert.Len(t, s.vmState, 1, "stale vm-a state should be pruned")
		assert.False(t, hasScheduleSlot(t, s, "ns1/vm-a"), "vm-a should no longer be scheduled")
		assert.True(t, hasScheduleSlot(t, s, "ns2/vm-b"))
	})

	t.Run("should reset scrape state when VM ID changes at the same key", func(t *testing.T) {
		const key = "ns1/vm-a"
		oldVM := &virtualmachine.Info{
			ID:        "uid-old",
			Namespace: "ns1",
			Name:      "vm-a",
			VSOCKCID:  new(uint32(100)),
			Running:   true,
		}
		store := &mockStore{vms: []*virtualmachine.Info{oldVM}}
		s, clock := newTestScraper(t, store, &mockDialer{}, &mockProtocolClient{
			resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
		})

		s.pollOnce(context.Background())
		require.Equal(t, "1", cachedToken(t, s, key))
		require.Equal(t, 1, s.Stats().VMsScanned)
		require.Equal(t, clock.Now().Add(s.interval), cachedNextAttemptAt(t, s, key),
			"successful scrape schedules the next poll")

		store.vms = []*virtualmachine.Info{{
			ID:        "uid-new",
			Namespace: "ns1",
			Name:      "vm-a",
			VSOCKCID:  new(uint32(101)),
			Running:   true,
		}}
		s.reconcile()

		require.Equal(t, virtualmachine.VMID("uid-new"), concurrency.WithLock1(&s.mu, func() virtualmachine.VMID {
			return s.vmState[key].vmID
		}))
		assert.Empty(t, cachedToken(t, s, key))
		assert.Equal(t, 1, s.Stats().TrackedVMs)
		assert.Zero(t, s.Stats().VMsScanned, "replacement must not inherit prior scrape state")
		assert.Equal(t, clock.Now(), cachedNextAttemptAt(t, s, key),
			"replacement must be due immediately, not on the prior poll timer")
		assert.Contains(t, s.dueKeys(), key)
	})
}

// TestVMScraper_IgnoresLateScrapeAfterVMReplacement covers a KubeVirt recreate
// while a scrape of the predecessor is still forwarding: the late commit must not
// mark the replacement as already scanned or push its next poll out by interval.
func TestVMScraper_IgnoresLateScrapeAfterVMReplacement(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const key = "ns1/vm-a"
		oldVM := &virtualmachine.Info{
			ID:        "uid-old",
			Namespace: "ns1",
			Name:      "vm-a",
			VSOCKCID:  new(uint32(100)),
			Running:   true,
		}
		store := &mockStore{vms: []*virtualmachine.Info{oldVM}}
		s, clock := newTestScraper(t, store, &mockDialer{}, &mockProtocolClient{
			resultQueue: []*vsockclient.GetReportResult{makeReport("99")},
		})
		s.toCentral = make(chan *message.ExpiringMessage)

		done := make(chan struct{})
		go func() {
			defer close(done)
			s.pollOnce(t.Context())
		}()
		synctest.Wait()

		store.vms = []*virtualmachine.Info{{
			ID:        "uid-new",
			Namespace: "ns1",
			Name:      "vm-a",
			VSOCKCID:  new(uint32(101)),
			Running:   true,
		}}
		s.reconcile()

		require.Equal(t, virtualmachine.VMID("uid-new"), concurrency.WithLock1(&s.mu, func() virtualmachine.VMID {
			return s.vmState[key].vmID
		}))
		require.Zero(t, s.Stats().VMsScanned)
		require.Equal(t, clock.Now(), cachedNextAttemptAt(t, s, key))

		<-s.toCentral
		<-done

		assert.Equal(t, virtualmachine.VMID("uid-new"), concurrency.WithLock1(&s.mu, func() virtualmachine.VMID {
			return s.vmState[key].vmID
		}))
		assert.Empty(t, cachedToken(t, s, key), "late commit must not stamp the replacement")
		assert.Zero(t, s.Stats().VMsScanned, "replacement must stay unscanned until its own scrape")
		assert.Equal(t, clock.Now(), cachedNextAttemptAt(t, s, key),
			"late schedule must not inherit the predecessor's poll timer")
		assert.Contains(t, s.dueKeys(), key)
	})
}

// hasScheduleSlot reports whether key has a vmState slot (reconcile membership).
func hasScheduleSlot(t *testing.T, s *VMScraper, key string) bool {
	t.Helper()
	return concurrency.WithLock1(&s.mu, func() bool {
		_, ok := s.vmState[key]
		return ok
	})
}

// cachedToken reads a VM's cached token under s.mu, so the read is
// race-safe regardless of what locking handleNACK or commitVMState use internally.
func cachedToken(t *testing.T, s *VMScraper, key string) string {
	t.Helper()
	return concurrency.WithLock1(&s.mu, func() string {
		st, ok := s.vmState[key]
		require.True(t, ok, "no cached state for %q", key)
		return st.lastToken
	})
}

type pullOutcomeSample struct {
	vec   *prometheus.CounterVec
	label string
}

func allPullOutcomeSamples() []pullOutcomeSample {
	var out []pullOutcomeSample
	for _, label := range []string{
		metrics.PullTransportDialError,
		metrics.PullTransportTimeout,
		metrics.PullTransportReadError,
		metrics.PullTransportAbnormalClose,
		metrics.PullTransportUnexpected,
	} {
		out = append(out, pullOutcomeSample{metrics.PullTransportTotal, label})
	}
	for _, label := range []string{
		metrics.PullGetReportUnchanged,
		metrics.PullGetReportNotReady,
		metrics.PullGetReportMappingRequired,
		metrics.PullGetReportUnknownMethod,
		metrics.PullGetReportBusy,
		metrics.PullGetReportInternalError,
		metrics.PullGetReportMalformedRequest,
		metrics.PullGetReportRequestTooLarge,
		metrics.PullGetReportUnknownAgentError,
	} {
		out = append(out, pullOutcomeSample{metrics.PullGetReportTotal, label})
	}
	for _, label := range []string{
		metrics.PullScrapeSuccess,
		metrics.PullScrapeInvalidReport,
		metrics.PullScrapeSendError,
	} {
		out = append(out, pullOutcomeSample{metrics.PullScrapeTotal, label})
	}
	return out
}

func snapshotPullOutcomes() map[pullOutcomeSample]float64 {
	snap := make(map[pullOutcomeSample]float64, len(allPullOutcomeSamples()))
	for _, sample := range allPullOutcomeSamples() {
		snap[sample] = testutil.ToFloat64(sample.vec.WithLabelValues(sample.label))
	}
	return snap
}

func assertOnlyPullOutcomeIncremented(t *testing.T, before map[pullOutcomeSample]float64, wantVec *prometheus.CounterVec, wantLabel string) {
	t.Helper()
	for _, sample := range allPullOutcomeSamples() {
		got := testutil.ToFloat64(sample.vec.WithLabelValues(sample.label))
		want := before[sample]
		if sample.vec == wantVec && sample.label == wantLabel {
			want++
		}
		assert.Equal(t, want, got, "status %q", sample.label)
	}
}

// TestHandleGetReportError_ClassifiesEveryErrorCode locks the mapping from
// each error to its metric label and retry classification.
func TestHandleGetReportError_ClassifiesEveryErrorCode(t *testing.T) {
	cases := map[string]struct {
		err         error
		wantMetric  *prometheus.CounterVec
		wantLabel   string
		wantOutcome scrapeOutcome
	}{
		"NOT_READY maps to get_report not_ready": {
			err:         fmt.Errorf("%w: still scanning", vsockclient.ErrNotReady),
			wantMetric:  metrics.PullGetReportTotal,
			wantLabel:   metrics.PullGetReportNotReady,
			wantOutcome: scrapeRetryable,
		},
		"MAPPING_REQUIRED maps to get_report mapping_required": {
			err:         fmt.Errorf("%w: no mapping yet", vsockclient.ErrMappingRequired),
			wantMetric:  metrics.PullGetReportTotal,
			wantLabel:   metrics.PullGetReportMappingRequired,
			wantOutcome: scrapeRetryable,
		},
		"UNKNOWN_METHOD maps to get_report unknown_method": {
			err:         fmt.Errorf("%w: no get_report", vsockclient.ErrUnknownMethod),
			wantMetric:  metrics.PullGetReportTotal,
			wantLabel:   metrics.PullGetReportUnknownMethod,
			wantOutcome: scrapeNonRetryable,
		},
		"BUSY maps to get_report busy": {
			err:         fmt.Errorf("%w: another request in flight", vsockclient.ErrBusy),
			wantMetric:  metrics.PullGetReportTotal,
			wantLabel:   metrics.PullGetReportBusy,
			wantOutcome: scrapeRetryable,
		},
		"INTERNAL maps to get_report internal_error": {
			err:         fmt.Errorf("%w: panic recovered", vsockclient.ErrInternal),
			wantMetric:  metrics.PullGetReportTotal,
			wantLabel:   metrics.PullGetReportInternalError,
			wantOutcome: scrapeRetryable,
		},
		"MALFORMED_REQUEST maps to get_report malformed_request": {
			err:         fmt.Errorf("%w: empty request_id", vsockclient.ErrMalformedRequest),
			wantMetric:  metrics.PullGetReportTotal,
			wantLabel:   metrics.PullGetReportMalformedRequest,
			wantOutcome: scrapeNonRetryable,
		},
		"REQUEST_TOO_LARGE maps to get_report request_too_large": {
			err:         fmt.Errorf("%w: payload too big", vsockclient.ErrRequestTooLarge),
			wantMetric:  metrics.PullGetReportTotal,
			wantLabel:   metrics.PullGetReportRequestTooLarge,
			wantOutcome: scrapeNonRetryable,
		},
		"unrecognized agent error code maps to get_report unknown_agent_error": {
			err:         fmt.Errorf("%w: agent error (99): ?", vsockclient.ErrUnknownAgentError),
			wantMetric:  metrics.PullGetReportTotal,
			wantLabel:   metrics.PullGetReportUnknownAgentError,
			wantOutcome: scrapeRetryable,
		},
		"abnormal websocket close maps to transport abnormal_close": {
			err:         fmt.Errorf("reading response: %w", &fakeCloseCoder{code: 1006, reason: "abnormal closure"}),
			wantMetric:  metrics.PullTransportTotal,
			wantLabel:   metrics.PullTransportAbnormalClose,
			wantOutcome: scrapeRetryable,
		},
		"plain io.EOF maps to transport read_error": {
			err:         io.EOF,
			wantMetric:  metrics.PullTransportTotal,
			wantLabel:   metrics.PullTransportReadError,
			wantOutcome: scrapeRetryable,
		},
		"io.ErrUnexpectedEOF maps to transport read_error": {
			err:         io.ErrUnexpectedEOF,
			wantMetric:  metrics.PullTransportTotal,
			wantLabel:   metrics.PullTransportReadError,
			wantOutcome: scrapeRetryable,
		},
		"unrecognized transport/framing error maps to transport unexpected": {
			err:         errors.New("unmarshaling response: unexpected EOF in the middle of a varint"),
			wantMetric:  metrics.PullTransportTotal,
			wantLabel:   metrics.PullTransportUnexpected,
			wantOutcome: scrapeRetryable,
		},
	}

	s := &VMScraper{}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			before := snapshotPullOutcomes()
			outcome := s.handleGetReportError(t.Context(), "ns/vm", tc.err)
			assert.Equal(t, tc.wantOutcome, outcome)
			assertOnlyPullOutcomeIncremented(t, before, tc.wantMetric, tc.wantLabel)
		})
	}
}

// TestIsAbnormalClose locks which close codes take the abnormal-close path
// versus the ordinary EOF path.
func TestIsAbnormalClose(t *testing.T) {
	cases := map[string]struct {
		err  error
		want bool
	}{
		"normal closure (1000) is not abnormal": {
			err:  &fakeCloseCoder{code: closeCodeNormalClosure, reason: "bye"},
			want: false,
		},
		"zero code (no structured signal) is not abnormal": {
			err:  &fakeCloseCoder{code: 0},
			want: false,
		},
		"abnormal closure (1006) is abnormal": {
			err:  &fakeCloseCoder{code: 1006, reason: "abnormal closure"},
			want: true,
		},
		"a plain io.EOF is not a closeCoder at all": {
			err:  io.EOF,
			want: false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var target closeCoder
			assert.Equal(t, tc.want, isAbnormalClose(tc.err, &target))
		})
	}
}

func drainToCentral(s *VMScraper) []*message.ExpiringMessage {
	var out []*message.ExpiringMessage
	for {
		select {
		case msg := <-s.toCentral:
			out = append(out, msg)
		default:
			return out
		}
	}
}

func forwardedCount(s *VMScraper) int {
	n := 0
	for _, msg := range drainToCentral(s) {
		if msg.GetEvent().GetVirtualMachineIndexReport() != nil {
			n++
		}
	}
	return n
}

func drainVMUpdates(s *VMScraper) []*pb.VirtualMachine {
	var out []*pb.VirtualMachine
	for _, msg := range drainToCentral(s) {
		if vm := msg.GetEvent().GetVirtualMachine(); vm != nil {
			out = append(out, vm)
		}
	}
	return out
}

type staticClusterID string

func (c staticClusterID) GetNoWait() string { return string(c) }

func newTestScraper(t *testing.T, store RunningVMStore, dialer VMDialer, client ProtocolClient) (*VMScraper, *testClock) {
	t.Helper()
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported})
	t.Cleanup(func() { centralcaps.Set(nil) })

	clock := newTestClock()
	interval := 5 * time.Minute
	s := &VMScraper{
		store:                 store,
		dialer:                dialer,
		client:                client,
		clusterID:             staticClusterID("test-cluster"),
		toCentral:             make(chan *message.ExpiringMessage, 256),
		centralReady:          concurrency.NewSignal(),
		interval:              interval,
		tickInterval:          defaultTickInterval,
		initialBackoff:        initialBackoff,
		reconcileEvery:        reconcilePeriod(interval),
		perVMTimeout:          10 * time.Second,
		mandatoryRefreshAfter: 4 * time.Hour,
		concurrency:           20,
		// Half of the 16MiB default pull response-size ceiling — same
		// derivation New() uses from env.VirtualMachinesPullMaxResponseSizeKB.
		warnMaxBytes:   8 << 20,
		spreadFraction: 2.0 / 3,
		vmState:        make(map[string]*vmState),
		inFlight:       set.NewStringSet(),
		now:            clock.Now,
		randFloat64:    func() float64 { return 0 },
	}
	s.centralReady.Signal()
	setTickToDrain(s)
	return s, clock
}

// pollOnce forces a reconcile and scrapes due slots (subject to the per-tick start cap).
func (s *VMScraper) pollOnce(ctx context.Context) {
	s.tick(ctx, true)
}

// setTickToDrain sets tickInterval to the new-VM index report window so one
// tick can start every never-scraped due VM under concurrency.
func setTickToDrain(s *VMScraper) {
	s.tickInterval = newVMIndexReportWindow(s.interval)
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
	token string
	calls int
}

func (c *safeProtocolClient) GetReport(_ context.Context, _ io.ReadWriteCloser, _ string) (*vsockclient.GetReportResult, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return makeReport(c.token), nil
}

func (c *safeProtocolClient) SyncRepoCPEMapping(_ context.Context, _ io.ReadWriteCloser, _ []byte) (bool, *pb.ResponseMeta, error) {
	return false, nil, nil
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
	dialer := &delayDialer{delay: dialDelay}
	client := &safeProtocolClient{token: "1"}

	s, _ := newTestScraper(t, store, dialer, client)
	s.concurrency = concurrency
	// This test measures real dial overlap via delayDialer's timers, so it
	// needs the wall clock rather than the fake test clock.
	s.now = time.Now

	s.pollOnce(context.Background())

	assert.Equal(t, numVMs, forwardedCount(s))
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
	s, clock := newTestScraper(t, store, &mockDialer{}, client)

	s.pollOnce(context.Background())
	require.Len(t, client.calls, 1)
	assert.Equal(t, initialBackoff, cachedBackoff(t, s, "ns1/vm-a"))

	client.reset()
	client.errQueue = nil
	client.resultQueue = []*vsockclient.GetReportResult{makeReport("1")}
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
		client          *mockProtocolClient
		centralNotReady bool
		wantBackoff     time.Duration
		wantGap         time.Duration
	}{
		"send failure should retry using backoff": {
			client:          &mockProtocolClient{resultQueue: []*vsockclient.GetReportResult{makeReport("1")}},
			centralNotReady: true,
			wantBackoff:     initialBackoff,
			wantGap:         initialBackoff,
		},
		"ErrInternal should retry using backoff": {
			client:      &mockProtocolClient{errQueue: []error{vsockclient.ErrInternal}},
			wantBackoff: initialBackoff,
			wantGap:     initialBackoff,
		},
		"ErrMappingRequired should retry using backoff": {
			client:      &mockProtocolClient{errQueue: []error{vsockclient.ErrMappingRequired}},
			wantBackoff: initialBackoff,
			wantGap:     initialBackoff,
		},
		"io.EOF should retry using backoff": {
			client:      &mockProtocolClient{errQueue: []error{io.EOF}},
			wantBackoff: initialBackoff,
			wantGap:     initialBackoff,
		},
		"io.ErrUnexpectedEOF should retry using backoff": {
			client:      &mockProtocolClient{errQueue: []error{io.ErrUnexpectedEOF}},
			wantBackoff: initialBackoff,
			wantGap:     initialBackoff,
		},
		"unhandled protocol error should retry using backoff": {
			client:      &mockProtocolClient{errQueue: []error{errors.New("bogus frame")}},
			wantBackoff: initialBackoff,
			wantGap:     initialBackoff,
		},
		"ErrUnknownMethod should not retry using backoff": {
			client:      &mockProtocolClient{errQueue: []error{vsockclient.ErrUnknownMethod}},
			wantBackoff: 0,
			wantGap:     5 * time.Minute, // matches newTestScraper interval
		},
		"ErrMalformedRequest should not retry using backoff": {
			client:      &mockProtocolClient{errQueue: []error{vsockclient.ErrMalformedRequest}},
			wantBackoff: 0,
			wantGap:     5 * time.Minute,
		},
		"ErrRequestTooLarge should not retry using backoff": {
			client:      &mockProtocolClient{errQueue: []error{vsockclient.ErrRequestTooLarge}},
			wantBackoff: 0,
			wantGap:     5 * time.Minute,
		},
		"invalid report should not retry using backoff": {
			client: &mockProtocolClient{resultQueue: []*vsockclient.GetReportResult{{
				Meta: &pb.ResponseMeta{ReportToken: "1"},
			}}},
			wantBackoff: 0,
			wantGap:     5 * time.Minute,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s, clock := newTestScraper(t,
				&mockStore{vms: []*virtualmachine.Info{makeVM("ns1", "vm-a", 100)}},
				&mockDialer{}, tc.client,
			)
			if tc.centralNotReady {
				s.centralReady.Reset()
			}
			start := clock.Now()
			s.pollOnce(context.Background())

			assert.Equal(t, tc.wantBackoff, cachedBackoff(t, s, key))
			assert.Equal(t, tc.wantGap, cachedNextAttemptAt(t, s, key).Sub(start))
		})
	}
}

func TestVMScraper_MaybeSyncRepoCPEMapping(t *testing.T) {
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
			wantMetric:    metrics.PullSyncSuccess,
		},
		"URL mismatch should log and count but never dial": {
			meta:       metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_URL),
			fetcher:    &fakeFetcher{ok: true, hash: "new-hash"},
			wantMetric: metrics.PullSyncURLHashMismatch,
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
			wantMetric:    metrics.PullSyncNotManaged,
		},
		"a generic sync failure should log and count": {
			meta:          metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR),
			fetcher:       &fakeFetcher{ok: true, hash: "new-hash", mapping: []byte("payload")},
			syncErr:       errors.New("connection reset"),
			wantSyncCalls: 1,
			wantMetric:    metrics.PullSyncError,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			client := &mockProtocolClient{syncErr: tc.syncErr}
			s, _ := newTestScraper(t, &mockStore{}, &mockDialer{}, client)
			if !tc.noFetcher {
				s.repo2CPEFetcher = tc.fetcher
			}

			var before float64
			if tc.wantMetric != "" {
				before = testutil.ToFloat64(metrics.PullSyncTotal.WithLabelValues(tc.wantMetric))
			}

			s.maybeSyncRepoCPEMapping(context.Background(), vm, "ns1/vm-a", 9999, tc.meta)

			assert.Len(t, client.syncCalls, tc.wantSyncCalls)
			if tc.wantMetric != "" {
				assert.Equal(t, before+1, testutil.ToFloat64(metrics.PullSyncTotal.WithLabelValues(tc.wantMetric)))
			}
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
		resultQueue: []*vsockclient.GetReportResult{makeReport("1"), makeReport("1")},
	}
	s, clock := newTestScraper(t, store, &mockDialer{}, client)

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

// TestVMScraper_DialAndGetReport_SyncTriggering exercises maybeSyncRepoCPEMapping's
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
		wantOutcome   scrapeOutcome
		wantSyncCalls int
	}{
		"MAPPING_REQUIRED error triggers a sync attempt": {
			resultQueue:   []*vsockclient.GetReportResult{{Meta: metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR)}},
			errQueue:      []error{vsockclient.ErrMappingRequired},
			wantOutcome:   scrapeRetryable,
			wantSyncCalls: 1,
		},
		"NOT_READY never triggers a sync attempt": {
			errQueue:    []error{vsockclient.ErrNotReady},
			wantOutcome: scrapeRetryable,
		},
		"a generic protocol error never triggers a sync attempt": {
			errQueue:    []error{errors.New("connection refused")},
			wantOutcome: scrapeRetryable,
		},
		"a successful report with a stale Sensor-managed hash also triggers a sync": {
			resultQueue: []*vsockclient.GetReportResult{{
				IndexReport: &v4.IndexReport{State: "IndexFinished"},
				Meta:        metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR),
			}},
			wantOutcome:   scrapeOK,
			wantSyncCalls: 1,
		},
		"an Unchanged report with a matching hash never triggers a sync": {
			resultQueue: []*vsockclient.GetReportResult{{
				Unchanged: true,
				Meta:      metaWithMapping("new-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR),
			}},
			wantOutcome: scrapeOK,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			client := &mockProtocolClient{resultQueue: tc.resultQueue, errQueue: tc.errQueue}
			s, _ := newTestScraper(t, &mockStore{}, &mockDialer{}, client)
			s.repo2CPEFetcher = staleFetcher()

			_, outcome := s.dialAndGetReport(context.Background(), vm, "ns1/vm-a", 1, "")

			assert.Equal(t, tc.wantOutcome, outcome)
			assert.Len(t, client.syncCalls, tc.wantSyncCalls)
		})
	}
}

// TestVMScraper_SyncRepoCPEMapping_CanceledParent_ClassifiedAsTimeout covers
// Sensor stop counting as PullSyncTimeout, not PullSyncError.
func TestVMScraper_SyncRepoCPEMapping_CanceledParent_ClassifiedAsTimeout(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)
	dialer := &mockDialer{err: errors.New("dial must not be attempted")}
	s, _ := newTestScraper(t, &mockStore{}, dialer, &mockProtocolClient{})

	timeoutBefore := testutil.ToFloat64(metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncTimeout))
	syncErrBefore := testutil.ToFloat64(metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncError))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.syncRepoCPEMapping(ctx, vm, "ns1/vm-a", 1, []byte("payload"))

	assert.Zero(t, dialer.callIdx.Load(), "a canceled parent must not be dialed at all")
	assert.Equal(t, timeoutBefore+1, testutil.ToFloat64(metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncTimeout)))
	assert.Equal(t, syncErrBefore, testutil.ToFloat64(metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncError)),
		"a canceled parent must not be counted as a sync error")
}

// TestVMScraper_SyncRepoCPEMapping_DialTimeout_ClassifiedAsTimeout covers
// perVMTimeout expiring during Dial counting as PullSyncTimeout, not
// PullSyncError (same rule as dialAndGetReport).
func TestVMScraper_SyncRepoCPEMapping_DialTimeout_ClassifiedAsTimeout(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)
	s, _ := newTestScraper(t, &mockStore{}, blockingDialer{}, &mockProtocolClient{})
	s.perVMTimeout = 20 * time.Millisecond

	timeoutBefore := testutil.ToFloat64(metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncTimeout))
	syncErrBefore := testutil.ToFloat64(metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncError))

	s.syncRepoCPEMapping(context.Background(), vm, "ns1/vm-a", 1, []byte("payload"))

	assert.Equal(t, timeoutBefore+1, testutil.ToFloat64(metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncTimeout)))
	assert.Equal(t, syncErrBefore, testutil.ToFloat64(metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncError)),
		"a dial timeout must not be counted as a sync error")
}

// TestVMScraper_SyncRepoCPEMapping_SyncTimeout_ClassifiedAsTimeout covers
// perVMTimeout expiring during SyncRepoCPEMapping counting as
// PullSyncTimeout, not PullSyncError (same rule as handleGetReportError).
func TestVMScraper_SyncRepoCPEMapping_SyncTimeout_ClassifiedAsTimeout(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)
	client := &mockProtocolClient{syncBlocksOnCtx: true}
	s, _ := newTestScraper(t, &mockStore{}, &mockDialer{}, client)
	s.perVMTimeout = 20 * time.Millisecond

	timeoutBefore := testutil.ToFloat64(metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncTimeout))
	syncErrBefore := testutil.ToFloat64(metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncError))

	s.syncRepoCPEMapping(context.Background(), vm, "ns1/vm-a", 1, []byte("payload"))

	require.Len(t, client.syncCalls, 1)
	assert.Equal(t, timeoutBefore+1, testutil.ToFloat64(metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncTimeout)))
	assert.Equal(t, syncErrBefore, testutil.ToFloat64(metrics.PullSyncTotal.WithLabelValues(metrics.PullSyncError)),
		"a sync timeout must not be counted as a sync error")
}

// TestVMScraper_SendSucceedsAfterSyncOverrunsGetReportDeadline covers Send
// succeeding after GetReport and mapping sync each consume a full timeout.
func TestVMScraper_SendSucceedsAfterSyncOverrunsGetReportDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		vm := makeVM("ns1", "vm-a", 100)
		rep := makeReport("1")
		rep.Meta.RepoCpeMappingHash = new("old-hash")
		rep.Meta.RepoCpeMappingUpdatePath = pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR.Enum()
		client := &mockProtocolClient{
			resultQueue:    []*vsockclient.GetReportResult{rep},
			getReportDelay: 15 * time.Millisecond,
			syncDelay:      15 * time.Millisecond,
		}
		s, _ := newTestScraper(t, &mockStore{vms: []*virtualmachine.Info{vm}}, &mockDialer{}, client)
		s.perVMTimeout = 20 * time.Millisecond
		s.repo2CPEFetcher = &fakeFetcher{ok: true, hash: "new-hash", mapping: []byte("payload")}

		s.pollOnce(t.Context())

		require.Equal(t, 1, forwardedCount(s), "GetReport's deadline must not fail forward after a mapping sync")
		require.Len(t, client.syncCalls, 1)
	})
}

// TestVMScraper_MappingRequiredClassifiedDespiteSlowSync covers MAPPING_REQUIRED
// staying mapping_required when mapping sync overruns GetReport's deadline.
func TestVMScraper_MappingRequiredClassifiedDespiteSlowSync(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		vm := makeVM("ns1", "vm-a", 100)
		client := &mockProtocolClient{
			resultQueue: []*vsockclient.GetReportResult{{
				Meta: metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR),
			}},
			errQueue:       []error{vsockclient.ErrMappingRequired},
			getReportDelay: 15 * time.Millisecond,
			syncDelay:      15 * time.Millisecond,
		}
		s, _ := newTestScraper(t, &mockStore{vms: []*virtualmachine.Info{vm}}, &mockDialer{}, client)
		s.perVMTimeout = 20 * time.Millisecond
		s.repo2CPEFetcher = &fakeFetcher{ok: true, hash: "new-hash", mapping: []byte("payload")}

		mappingBefore := testutil.ToFloat64(metrics.PullGetReportTotal.WithLabelValues(metrics.PullGetReportMappingRequired))
		timeoutBefore := testutil.ToFloat64(metrics.PullTransportTotal.WithLabelValues(metrics.PullTransportTimeout))

		s.pollOnce(t.Context())

		require.Len(t, client.syncCalls, 1)
		assert.Equal(t, mappingBefore+1, testutil.ToFloat64(metrics.PullGetReportTotal.WithLabelValues(metrics.PullGetReportMappingRequired)))
		assert.Equal(t, timeoutBefore, testutil.ToFloat64(metrics.PullTransportTotal.WithLabelValues(metrics.PullTransportTimeout)),
			"a slow mapping sync must not rewrite MAPPING_REQUIRED as a transport timeout")
	})
}

// TestVMScraper_DialAndGetReport_ClosesConnectionBeforeSync guards against
// maybeSyncRepoCPEMapping's second dial racing the first GetReport connection: roxagent
// allows only one connection at a time, so dialAndGetReport must close its
// stream before calling maybeSyncRepoCPEMapping, not rely solely on its deferred close.
func TestVMScraper_DialAndGetReport_ClosesConnectionBeforeSync(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)
	dialer := &singleConnDialer{}
	client := &mockProtocolClient{resultQueue: []*vsockclient.GetReportResult{{
		IndexReport: &v4.IndexReport{State: "IndexFinished"},
		Meta:        metaWithMapping("old-hash", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR),
	}}}
	s, _ := newTestScraper(t, &mockStore{}, dialer, client)
	s.repo2CPEFetcher = &fakeFetcher{ok: true, hash: "new-hash", mapping: []byte("payload")}

	_, outcome := s.dialAndGetReport(context.Background(), vm, "ns1/vm-a", 1, "")

	require.Equal(t, scrapeOK, outcome)
	require.Len(t, client.syncCalls, 1, "maybeSyncRepoCPEMapping's dial must succeed, not be rejected as busy by a still-open first connection")
}

// TestVMScraper_DialAndGetReport_MappingRequired_ClosesConnectionBeforeSync
// is the MAPPING_REQUIRED-error counterpart: that path closes and syncs
// from a different branch than the success path above.
func TestVMScraper_DialAndGetReport_MappingRequired_ClosesConnectionBeforeSync(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)
	dialer := &singleConnDialer{}
	client := &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{{Meta: metaWithMapping("", pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR)}},
		errQueue:    []error{vsockclient.ErrMappingRequired},
	}
	s, _ := newTestScraper(t, &mockStore{}, dialer, client)
	s.repo2CPEFetcher = &fakeFetcher{ok: true, hash: "new-hash", mapping: []byte("payload")}

	_, outcome := s.dialAndGetReport(context.Background(), vm, "ns1/vm-a", 1, "")

	require.Equal(t, scrapeRetryable, outcome)
	require.Len(t, client.syncCalls, 1, "maybeSyncRepoCPEMapping's dial must succeed, not be rejected as busy by a still-open first connection")
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

func TestVMScraper_Capabilities(t *testing.T) {
	s, _ := newTestScraper(t, &mockStore{}, &mockDialer{}, &mockProtocolClient{})
	assert.Equal(t, []centralsensor.SensorCapability{centralsensor.SensorACKSupport}, s.Capabilities())
}

func TestVMScraper_ForwardReport(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)
	s, _ := newTestScraper(t, &mockStore{vms: []*virtualmachine.Info{vm}}, &mockDialer{}, &mockProtocolClient{})
	require.NoError(t, s.forwardReport(t.Context(), vm, makeReport("tok-1").IndexReport))
	assert.Equal(t, 1, forwardedCount(s))
}

func TestVMScraper_ForwardReport_EnqueueBlocked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		vm := makeVM("ns1", "vm-a", 100)
		s, _ := newTestScraper(t, &mockStore{vms: []*virtualmachine.Info{vm}}, &mockDialer{}, &mockProtocolClient{})
		s.toCentral = make(chan *message.ExpiringMessage)
		blockedBefore := testutil.ToFloat64(metrics.IndexReportEnqueueBlockedTotal)
		done := make(chan error, 1)
		go func() {
			done <- s.forwardReport(t.Context(), vm, makeReport("tok-1").IndexReport)
		}()
		synctest.Wait()
		assert.Equal(t, blockedBefore+1, testutil.ToFloat64(metrics.IndexReportEnqueueBlockedTotal))
		<-s.toCentral
		require.NoError(t, <-done)
	})
}
