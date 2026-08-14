package vmscraper

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingDialer counts Dial calls for start-budget assertions.
type recordingDialer struct {
	calls atomic.Int32
}

func (d *recordingDialer) Dial(_ context.Context, _, _ string, _ uint32, _ bool) (io.ReadWriteCloser, error) {
	d.calls.Add(1)
	return nopCloser{}, nil
}

// sequencedRand returns successive values from seq, then the last value.
func sequencedRand(seq []float64) func() float64 {
	var i atomic.Int32
	return func() float64 {
		idx := int(i.Add(1) - 1)
		if idx >= len(seq) {
			return seq[len(seq)-1]
		}
		return seq[idx]
	}
}

func TestVMScraper_CadenceRescheduleUsesSteadyBand(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{makeVM("ns", "vm-a", 1)}}
	s, clock := newTestScraper(store, &mockSender{}, &mockDialer{}, &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport(1)},
	})
	s.spreadFraction = 2.0 / 3
	// First draw (reconcile insert) keeps the VM due now; second is the cadence offset.
	s.randFloat64 = sequencedRand([]float64{0, 0.5})

	start := clock.Now()
	s.pollOnce(context.Background())

	steadyWidth := steadySpreadWidth(s.interval, s.spreadFraction)
	want := start.Add(s.interval + steadyWidth/2)
	assert.Equal(t, want, cachedNextAttemptAt(t, s, "ns/vm-a"))
	assert.Greater(t, histogramSampleCount(t, metrics.PullScheduleOffsetSeconds), uint64(0))
}

func TestVMScraper_ManyCadencedSuccessesSpanSteadyBand(t *testing.T) {
	const numVMs = 5
	vms := make([]*virtualmachine.Info, 0, numVMs)
	for i := range numVMs {
		vms = append(vms, makeVM("ns", fmt.Sprintf("vm-%d", i), uint32(10+i)))
	}
	store := &mockStore{vms: vms}
	s, clock := newTestScraper(store, &safeSender{}, &recordingDialer{}, &safeProtocolClient{gen: 1})
	s.concurrency = numVMs
	s.interval = time.Hour
	s.spreadFraction = 2.0 / 3
	// Drain the urgent due pile in one tick; RNG is only used on cadence write.
	s.tickInterval = catchUpWindow(s.interval)
	s.reconcileEvery = reconcilePeriod(s.interval)
	s.randFloat64 = sequencedRand([]float64{0, 0.25, 0.5, 0.75, 1})

	now := clock.Now()
	concurrency.WithLock(&s.mu, func() {
		for _, vm := range vms {
			key := vm.Key()
			s.vmState[key] = &vmState{
				vmID:          vm.ID,
				nextAttemptAt: now,
				orderHash:     hashVMID(vm.ID, key),
			}
		}
		s.lastReconcile = now
	})

	s.tick(context.Background(), false)

	steadyWidth := steadySpreadWidth(s.interval, s.spreadFraction)
	minNext, maxNext := now.Add(s.interval), now.Add(s.interval+steadyWidth)
	var earliest, latest time.Time
	concurrency.WithLock(&s.mu, func() {
		for _, st := range s.vmState {
			if earliest.IsZero() || st.nextAttemptAt.Before(earliest) {
				earliest = st.nextAttemptAt
			}
			if latest.IsZero() || st.nextAttemptAt.After(latest) {
				latest = st.nextAttemptAt
			}
			assert.False(t, st.nextAttemptAt.Before(minNext))
			assert.False(t, st.nextAttemptAt.After(maxNext))
		}
	})
	assert.GreaterOrEqual(t, latest.Sub(earliest), steadyWidth/2,
		"distinct RNG draws should spread nextAttemptAt across a wide band")
}

func TestVMScraper_MassFirstInsertSpansCatchUpWindow(t *testing.T) {
	const numVMs = 10
	vms := make([]*virtualmachine.Info, 0, numVMs)
	for i := range numVMs {
		vms = append(vms, makeVM("ns", fmt.Sprintf("vm-%d", i), uint32(100+i)))
	}
	store := &mockStore{vms: vms}
	s, clock := newTestScraper(store, &mockSender{}, &recordingDialer{}, &safeProtocolClient{gen: 1})
	s.concurrency = numVMs
	s.interval = time.Hour
	units := make([]float64, numVMs)
	for i := range units {
		units[i] = float64(i) / float64(numVMs-1)
	}
	s.randFloat64 = sequencedRand(units)

	s.reconcile()

	catchUp := catchUpWindow(s.interval)
	now := clock.Now()
	var earliest, latest time.Time
	concurrency.WithLock(&s.mu, func() {
		require.Len(t, s.vmState, numVMs)
		for _, st := range s.vmState {
			assert.False(t, st.nextAttemptAt.Before(now))
			assert.False(t, st.nextAttemptAt.After(now.Add(catchUp)))
			if earliest.IsZero() || st.nextAttemptAt.Before(earliest) {
				earliest = st.nextAttemptAt
			}
			if latest.IsZero() || st.nextAttemptAt.After(latest) {
				latest = st.nextAttemptAt
			}
		}
	})
	assert.Greater(t, latest.Sub(earliest), time.Duration(0),
		"mass insert must not schedule every VM at exactly now")
	assert.GreaterOrEqual(t, latest.Sub(earliest), catchUp/2)
}

func TestVMScraper_SingleInsertUsesCatchUpSpread(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{makeVM("ns", "only", 1)}}
	s, clock := newTestScraper(store, &mockSender{}, &recordingDialer{}, &safeProtocolClient{gen: 1})
	s.concurrency = 20
	s.randFloat64 = func() float64 { return 0.5 }

	s.reconcile()

	catchUp := catchUpWindow(s.interval)
	assert.Equal(t, clock.Now().Add(catchUp/2), cachedNextAttemptAt(t, s, "ns/only"),
		"every first schedule uses catchUp, including a lone insert with free slots")
}

func TestVMScraper_MultiInsertUsesCatchUpSpread(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns", "a", 1),
		makeVM("ns", "b", 2),
	}}
	s, clock := newTestScraper(store, &mockSender{}, &recordingDialer{}, &safeProtocolClient{gen: 1})
	s.concurrency = 20
	s.randFloat64 = sequencedRand([]float64{0.25, 0.75})

	s.reconcile()

	catchUp := catchUpWindow(s.interval)
	assert.Equal(t, clock.Now().Add(catchUp/4), cachedNextAttemptAt(t, s, "ns/a"))
	assert.Equal(t, clock.Now().Add(3*catchUp/4), cachedNextAttemptAt(t, s, "ns/b"))
}

func TestVMScraper_UrgentDuePileIsPacedByCatchUpBudget(t *testing.T) {
	const numVMs = 10
	vms := make([]*virtualmachine.Info, 0, numVMs)
	for i := range numVMs {
		vms = append(vms, makeVM("ns", fmt.Sprintf("vm-%d", i), uint32(100+i)))
	}
	dialer := &recordingDialer{}
	s, _ := newTestScraper(&mockStore{vms: vms}, &mockSender{}, dialer, &safeProtocolClient{gen: 1})
	s.concurrency = numVMs
	s.interval = time.Hour
	s.tickInterval = defaultTickInterval
	s.reconcileEvery = reconcilePeriod(s.interval)

	now := s.now()
	concurrency.WithLock(&s.mu, func() {
		for _, vm := range vms {
			key := vm.Key()
			s.vmState[key] = &vmState{
				vmID:          vm.ID,
				nextAttemptAt: now,
				orderHash:     hashVMID(vm.ID, key),
			}
		}
		s.lastReconcile = now
	})

	catchUp := catchUpWindow(s.interval)
	wantBudget := startBudget(numVMs, s.tickInterval, catchUp)
	require.Equal(t, 1, wantBudget)
	require.Less(t, wantBudget, s.concurrency)

	s.tick(context.Background(), false)
	assert.Equal(t, int32(wantBudget), dialer.calls.Load())
	assert.Equal(t, float64(numVMs), testutil.ToFloat64(metrics.PullDueVMs))

	dialer.calls.Store(0)
	s.tick(context.Background(), false)
	assert.Equal(t, int32(wantBudget), dialer.calls.Load(), "leftovers stay due for later ticks")
}

func TestVMScraper_UrgentPreferredOverCadenced(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns", "old-a", 1),
		makeVM("ns", "old-b", 2),
		makeVM("ns", "new-x", 3),
		makeVM("ns", "new-y", 4),
	}}
	dialer := &recordingDialer{}
	s, _ := newTestScraper(store, &mockSender{}, dialer, &safeProtocolClient{gen: 1})
	s.concurrency = 2
	s.interval = time.Hour
	s.tickInterval = defaultTickInterval

	now := s.now()
	concurrency.WithLock(&s.mu, func() {
		s.vmState["ns/old-a"] = &vmState{
			vmID:            virtualmachine.VMID("ns/old-a"),
			lastForwardedAt: now.Add(-time.Hour),
			nextAttemptAt:   now,
			orderHash:       hashVMID(virtualmachine.VMID("ns/old-a"), "ns/old-a"),
		}
		s.vmState["ns/old-b"] = &vmState{
			vmID:            virtualmachine.VMID("ns/old-b"),
			lastForwardedAt: now.Add(-time.Hour),
			nextAttemptAt:   now,
			orderHash:       hashVMID(virtualmachine.VMID("ns/old-b"), "ns/old-b"),
		}
		s.vmState["ns/new-x"] = &vmState{
			vmID:          virtualmachine.VMID("ns/new-x"),
			nextAttemptAt: now,
			orderHash:     hashVMID(virtualmachine.VMID("ns/new-x"), "ns/new-x"),
		}
		s.vmState["ns/new-y"] = &vmState{
			vmID:          virtualmachine.VMID("ns/new-y"),
			nextAttemptAt: now,
			orderHash:     hashVMID(virtualmachine.VMID("ns/new-y"), "ns/new-y"),
		}
		s.lastReconcile = now
	})

	// nUrgent=2, catchUp=20m, production tick → budget=1; never-scraped must win the slot.
	s.tick(context.Background(), false)
	require.Equal(t, int32(1), dialer.calls.Load())

	dueOld := 0
	concurrency.WithLock(&s.mu, func() {
		for _, key := range []string{"ns/old-a", "ns/old-b"} {
			if !s.vmState[key].nextAttemptAt.After(s.now()) {
				dueOld++
			}
		}
	})
	assert.Equal(t, 2, dueOld, "cadenced VMs remain due when urgent fills the budget")
}

func TestVMScraper_CadencedDuePileUsesSteadyBudget(t *testing.T) {
	const numVMs = 10
	vms := make([]*virtualmachine.Info, 0, numVMs)
	for i := range numVMs {
		vms = append(vms, makeVM("ns", fmt.Sprintf("vm-%d", i), uint32(100+i)))
	}
	dialer := &recordingDialer{}
	s, _ := newTestScraper(&mockStore{vms: vms}, &mockSender{}, dialer, &safeProtocolClient{gen: 1})
	s.concurrency = numVMs
	s.interval = time.Hour
	s.spreadFraction = 2.0 / 3
	s.tickInterval = defaultTickInterval

	now := s.now()
	concurrency.WithLock(&s.mu, func() {
		for _, vm := range vms {
			key := vm.Key()
			s.vmState[key] = &vmState{
				vmID:            vm.ID,
				lastForwardedAt: now.Add(-time.Hour),
				nextAttemptAt:   now,
				orderHash:       hashVMID(vm.ID, key),
			}
		}
		s.lastReconcile = now
	})

	steadyWidth := steadySpreadWidth(s.interval, s.spreadFraction)
	wantBudget := startBudget(numVMs, s.tickInterval, steadyWidth)
	require.Equal(t, 1, wantBudget)

	s.tick(context.Background(), false)
	assert.Equal(t, int32(wantBudget), dialer.calls.Load())
	assert.Equal(t, float64(numVMs), testutil.ToFloat64(metrics.PullDueVMs))

	dialer.calls.Store(0)
	s.tick(context.Background(), false)
	assert.Equal(t, int32(wantBudget), dialer.calls.Load(), "leftovers stay due for later ticks")
}

func TestVMScraper_RetryPathDoesNotUseSteadyBand(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{makeVM("ns1", "vm-a", 100)}}
	client := &mockProtocolClient{errQueue: []error{vsockclient.ErrNotReady}}
	s, clock := newTestScraper(store, &mockSender{}, &mockDialer{}, client)
	// Insert due now; a later unit of 1 would be full steadyWidth if cadence used RNG.
	s.randFloat64 = sequencedRand([]float64{0, 1})

	start := clock.Now()
	s.pollOnce(context.Background())
	assert.Equal(t, initialBackoff, cachedNextAttemptAt(t, s, "ns1/vm-a").Sub(start),
		"retryable path must use short backoff, not pollInterval+steadyWidth")
	assert.Equal(t, initialBackoff, cachedBackoff(t, s, "ns1/vm-a"))
}

func TestVMScraper_NACKPathDoesNotUseSteadyBand(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)
	store := &mockStore{vms: []*virtualmachine.Info{vm}}
	s, clock := newTestScraper(store, &mockSender{}, &mockDialer{}, &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport(1)},
	})
	s.randFloat64 = sequencedRand([]float64{0, 1})
	s.pollOnce(context.Background())

	start := clock.Now()
	s.handleNACK(string(vm.ID) + ":1")
	assert.Equal(t, initialBackoff, cachedNextAttemptAt(t, s, "ns1/vm-a").Sub(start))
	assert.Equal(t, initialBackoff, cachedBackoff(t, s, "ns1/vm-a"))
}

func TestVMScraper_ForwardInterarrivalObservesAfterFirst(t *testing.T) {
	s, _ := newTestScraper(&mockStore{}, &mockSender{}, &mockDialer{}, &mockProtocolClient{})
	before := histogramSampleCount(t, metrics.PullForwardInterarrivalSeconds)
	s.observeForwardInterarrival()
	assert.Equal(t, before, histogramSampleCount(t, metrics.PullForwardInterarrivalSeconds),
		"first forward does not observe a gap")
	s.observeForwardInterarrival()
	assert.Equal(t, before+1, histogramSampleCount(t, metrics.PullForwardInterarrivalSeconds))
}

func TestSpreadFractionEnvDefault(t *testing.T) {
	assert.InDelta(t, 2.0/3, env.VirtualMachinesScraperSteadySpreadFraction.DefaultValue(), 1e-9)
	assert.InDelta(t, 2.0/3, env.VirtualMachinesScraperSteadySpreadFraction.FloatSetting(), 1e-9)
}

func histogramSampleCount(t *testing.T, h prometheus.Histogram) uint64 {
	t.Helper()
	var m dto.Metric
	require.NoError(t, h.(prometheus.Metric).Write(&m))
	return m.GetHistogram().GetSampleCount()
}
