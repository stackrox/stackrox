package vmscraper

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// recordingDialer counts Dial calls so tests can assert how many scrapes a tick started.
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
	s, clock := newTestScraper(t, store, &mockDialer{}, &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
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
	s, clock := newTestScraper(t, store, &mockDialer{}, &safeProtocolClient{token: "1"})
	s.concurrency = numVMs
	s.interval = time.Hour
	s.spreadFraction = 2.0 / 3
	s.tickInterval = newVMIndexReportWindow(s.interval)
	s.reconcileEvery = reconcilePeriod(s.interval)
	s.randFloat64 = sequencedRand([]float64{0, 0.25, 0.5, 0.75, 1})

	now := clock.Now()
	concurrency.WithLock(&s.mu, func() {
		for _, vm := range vms {
			s.vmState[vm.Key()] = &vmState{
				vmID:          vm.ID,
				nextAttemptAt: now,
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

func TestVMScraper_MassFirstInsertSpansNewVMIndexReportWindow(t *testing.T) {
	const numVMs = 10
	vms := make([]*virtualmachine.Info, 0, numVMs)
	for i := range numVMs {
		vms = append(vms, makeVM("ns", fmt.Sprintf("vm-%d", i), uint32(100+i)))
	}
	store := &mockStore{vms: vms}
	s, clock := newTestScraper(t, store, &mockDialer{}, &safeProtocolClient{token: "1"})
	s.concurrency = numVMs
	s.interval = time.Hour
	units := make([]float64, numVMs)
	for i := range units {
		units[i] = float64(i) / float64(numVMs-1)
	}
	s.randFloat64 = sequencedRand(units)

	s.reconcile()

	newVMWindow := newVMIndexReportWindow(s.interval)
	now := clock.Now()
	var earliest, latest time.Time
	concurrency.WithLock(&s.mu, func() {
		require.Len(t, s.vmState, numVMs)
		for _, st := range s.vmState {
			assert.False(t, st.nextAttemptAt.Before(now))
			assert.False(t, st.nextAttemptAt.After(now.Add(newVMWindow)))
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
	assert.GreaterOrEqual(t, latest.Sub(earliest), newVMWindow/2)
}

func TestVMScraper_SingleInsertUsesNewVMIndexReportSpread(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{makeVM("ns", "only", 1)}}
	s, clock := newTestScraper(t, store, &mockDialer{}, &safeProtocolClient{token: "1"})
	s.concurrency = 20
	s.randFloat64 = func() float64 { return 0.5 }

	s.reconcile()

	newVMWindow := newVMIndexReportWindow(s.interval)
	assert.Equal(t, clock.Now().Add(newVMWindow/2), cachedNextAttemptAt(t, s, "ns/only"),
		"every first schedule uses the new-VM index report window, including a lone insert with free slots")
}

func TestVMScraper_MultiInsertUsesNewVMIndexReportSpread(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{
		makeVM("ns", "a", 1),
		makeVM("ns", "b", 2),
	}}
	s, clock := newTestScraper(t, store, &mockDialer{}, &safeProtocolClient{token: "1"})
	s.concurrency = 20
	s.randFloat64 = sequencedRand([]float64{0.25, 0.75})

	s.reconcile()

	newVMWindow := newVMIndexReportWindow(s.interval)
	assert.Equal(t, clock.Now().Add(newVMWindow/4), cachedNextAttemptAt(t, s, "ns/a"))
	assert.Equal(t, clock.Now().Add(3*newVMWindow/4), cachedNextAttemptAt(t, s, "ns/b"))
}

func TestVMScraper_RetryPathDoesNotUseSteadyBand(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{makeVM("ns1", "vm-a", 100)}}
	client := &mockProtocolClient{errQueue: []error{vsockclient.ErrNotReady}}
	s, clock := newTestScraper(t, store, &mockDialer{}, client)
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
	s, clock := newTestScraper(t, store, &mockDialer{}, &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport("1")},
	})
	s.randFloat64 = sequencedRand([]float64{0, 1})
	s.pollOnce(context.Background())

	start := clock.Now()
	s.handleNACK(string(vm.ID) + ":1")
	assert.Equal(t, initialBackoff, cachedNextAttemptAt(t, s, "ns1/vm-a").Sub(start))
	assert.Equal(t, initialBackoff, cachedBackoff(t, s, "ns1/vm-a"))
}

func TestVMScraper_DuePileIsPacedByStartBudget(t *testing.T) {
	const numVMs = 10
	vms := make([]*virtualmachine.Info, 0, numVMs)
	for i := range numVMs {
		vms = append(vms, makeVM("ns", fmt.Sprintf("vm-%d", i), uint32(100+i)))
	}
	dialer := &recordingDialer{}
	s, _ := newTestScraper(t, &mockStore{vms: vms}, dialer, &safeProtocolClient{token: "1"})
	s.concurrency = numVMs
	s.interval = time.Hour
	s.tickInterval = defaultTickInterval
	s.reconcileEvery = reconcilePeriod(s.interval)

	now := s.now()
	concurrency.WithLock(&s.mu, func() {
		for _, vm := range vms {
			s.vmState[vm.Key()] = &vmState{
				vmID:          vm.ID,
				nextAttemptAt: now,
			}
		}
		s.lastReconcile = now
	})

	window := newVMIndexReportWindow(s.interval)
	want := startBudget(numVMs, s.tickInterval, window)
	require.Equal(t, 1, want)
	require.Greater(t, want, 0)
	require.Less(t, want, numVMs)
	require.LessOrEqual(t, want, s.concurrency)

	s.tick(context.Background(), false)
	assert.Equal(t, int32(want), dialer.calls.Load())

	dialer.calls.Store(0)
	s.tick(context.Background(), false)
	assert.Equal(t, int32(want), dialer.calls.Load(), "leftovers stay due for later ticks")
}

// TestVMScraper_AllDueClumpDrainsWithinIndexWindow covers a 100-VM all-due
// clump: fleet-sized budget drains it in one index window.
func TestVMScraper_AllDueClumpDrainsWithinIndexWindow(t *testing.T) {
	const numVMs = 100
	vms := make([]*virtualmachine.Info, 0, numVMs)
	for i := range numVMs {
		vms = append(vms, makeVM("ns", fmt.Sprintf("vm-%d", i), uint32(100+i)))
	}
	dialer := &recordingDialer{}
	s, _ := newTestScraper(t, &mockStore{vms: vms}, dialer, &safeProtocolClient{token: "1"})
	s.concurrency = numVMs
	s.tickInterval = defaultTickInterval
	s.reconcileEvery = reconcilePeriod(s.interval)

	now := s.now()
	concurrency.WithLock(&s.mu, func() {
		for _, vm := range vms {
			s.vmState[vm.Key()] = &vmState{
				vmID:          vm.ID,
				nextAttemptAt: now,
			}
		}
		s.lastReconcile = now
	})

	window := newVMIndexReportWindow(s.interval)
	require.Equal(t, 100*time.Second, window)
	perTick := startBudget(numVMs, s.tickInterval, window)
	require.Equal(t, 10, perTick, "ceil(100 × 10s / 100s) = 10")
	ticks := (numVMs + perTick - 1) / perTick
	require.Equal(t, 10, ticks)

	for range ticks {
		s.tick(context.Background(), false)
	}
	assert.Equal(t, int32(numVMs), dialer.calls.Load(),
		"a shrinking-n budget would start only 55 of 100 in 10 ticks (10+9+…+1)")
	assert.Empty(t, s.dueKeys(), "the clump should have left the due pile within the index window")
}

// TestVMScraper_SpreadDuePileStartsAtFleetRate covers VMs already spread by
// nextAttemptAt: the due snapshot is smaller than the fleet, but the rate
// must still be the fleet cap so every currently-due VM can start.
func TestVMScraper_SpreadDuePileStartsAtFleetRate(t *testing.T) {
	const (
		numVMs = 100
		nDue   = 10
	)
	vms := make([]*virtualmachine.Info, 0, numVMs)
	for i := range numVMs {
		vms = append(vms, makeVM("ns", fmt.Sprintf("vm-%d", i), uint32(100+i)))
	}
	dialer := &recordingDialer{}
	s, _ := newTestScraper(t, &mockStore{vms: vms}, dialer, &safeProtocolClient{token: "1"})
	s.concurrency = numVMs
	s.tickInterval = defaultTickInterval
	s.reconcileEvery = reconcilePeriod(s.interval)

	now := s.now()
	concurrency.WithLock(&s.mu, func() {
		for i, vm := range vms {
			st := &vmState{
				vmID:          vm.ID,
				nextAttemptAt: now.Add(time.Hour),
			}
			if i < nDue {
				st.nextAttemptAt = now
			}
			s.vmState[vm.Key()] = st
		}
		s.lastReconcile = now
	})
	require.Len(t, s.dueKeys(), nDue)

	s.tick(context.Background(), false)
	assert.Equal(t, int32(nDue), dialer.calls.Load(),
		"a due-pile budget would start 1 (ceil(10 × 10s / 100s)), not all 10 due")
	assert.Empty(t, s.dueKeys())
}

func TestSpreadFractionEnvDefault(t *testing.T) {
	assert.InDelta(t, 2.0/3, env.VirtualMachinesScraperSteadySpreadFraction.DefaultValue(), 1e-9)
	assert.InDelta(t, 2.0/3, env.VirtualMachinesScraperSteadySpreadFraction.FloatSetting(), 1e-9)
}

func TestVMScraper_StartsPerTickObservesLaunchCount(t *testing.T) {
	const numVMs = 5
	vms := make([]*virtualmachine.Info, 0, numVMs)
	for i := range numVMs {
		vms = append(vms, makeVM("ns", fmt.Sprintf("vm-%d", i), uint32(10+i)))
	}
	s, clock := newTestScraper(t, &mockStore{vms: vms}, &mockDialer{}, &safeProtocolClient{token: "1"})
	s.concurrency = numVMs
	s.reconcileEvery = reconcilePeriod(s.interval)

	now := clock.Now()
	concurrency.WithLock(&s.mu, func() {
		for _, vm := range vms {
			s.vmState[vm.Key()] = &vmState{
				vmID:          vm.ID,
				nextAttemptAt: now,
			}
		}
		s.lastReconcile = now
	})

	beforeCount := histogramSampleCount(t, metrics.PullStartsPerTick)
	beforeSum := histogramSampleSum(t, metrics.PullStartsPerTick)
	s.tick(context.Background(), false)
	assert.Equal(t, beforeCount+1, histogramSampleCount(t, metrics.PullStartsPerTick))
	assert.InDelta(t, beforeSum+float64(numVMs), histogramSampleSum(t, metrics.PullStartsPerTick), 1e-9)
}

func TestVMScraper_StartsPerTickSkipsIdleTicks(t *testing.T) {
	s, clock := newTestScraper(t, &mockStore{}, &mockDialer{}, &mockProtocolClient{})
	now := clock.Now()
	concurrency.WithLock(&s.mu, func() {
		s.lastReconcile = now
	})

	before := histogramSampleCount(t, metrics.PullStartsPerTick)
	s.tick(context.Background(), false)
	assert.Equal(t, before, histogramSampleCount(t, metrics.PullStartsPerTick),
		"idle ticks must not observe starts-per-tick")
}

func TestVMScraper_WarnIfSpreadSaturated(t *testing.T) {
	// newTestScraper uses a 5m poll, 2/3 spread, 10s tick (capacity 20).
	cases := map[string]struct {
		numVMs         int
		tickInterval   time.Duration
		spreadFraction float64
		wantLogs       int
	}{
		"under capacity does not warn": {
			numVMs:         20,
			tickInterval:   defaultTickInterval,
			spreadFraction: 2.0 / 3,
		},
		"over capacity warns": {
			numVMs:         21,
			tickInterval:   defaultTickInterval,
			spreadFraction: 2.0 / 3,
			wantLogs:       1,
		},
		"zero tick does not warn": {
			numVMs:         100,
			tickInterval:   time.Duration(0),
			spreadFraction: 2.0 / 3,
		},
		"zero spread does not warn": {
			numVMs:         100,
			tickInterval:   defaultTickInterval,
			spreadFraction: 0,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Must not use t.Parallel() because it temporarily modifies the package logger.
			core, logs := observer.New(zap.WarnLevel)
			orig := log
			log = &logging.LoggerImpl{InnerLogger: zap.New(core).Sugar()}
			t.Cleanup(func() { log = orig })

			s, _ := newTestScraper(t, &mockStore{}, &mockDialer{}, &mockProtocolClient{})
			s.tickInterval = tc.tickInterval
			s.spreadFraction = tc.spreadFraction
			s.warnIfSpreadSaturated(tc.numVMs)
			// Counting the number of Warn logs emitted to not assert on the log msg text.
			assert.Equal(t, tc.wantLogs, logs.Len())
		})
	}
}

func TestVMScraper_ForwardInterarrivalObservesAfterFirst(t *testing.T) {
	s, _ := newTestScraper(t, &mockStore{}, &mockDialer{}, &mockProtocolClient{})
	before := histogramSampleCount(t, metrics.PullForwardInterarrivalSeconds)
	s.observeForwardInterarrival()
	assert.Equal(t, before, histogramSampleCount(t, metrics.PullForwardInterarrivalSeconds),
		"first forward does not observe a gap")
	s.observeForwardInterarrival()
	assert.Equal(t, before+1, histogramSampleCount(t, metrics.PullForwardInterarrivalSeconds))
}

func histogramSampleCount(t *testing.T, h prometheus.Histogram) uint64 {
	t.Helper()
	return histogramMetric(t, h).GetSampleCount()
}

func histogramSampleSum(t *testing.T, h prometheus.Histogram) float64 {
	t.Helper()
	return histogramMetric(t, h).GetSampleSum()
}

func histogramMetric(t *testing.T, h prometheus.Histogram) *dto.Histogram {
	t.Helper()
	var m dto.Metric
	require.NoError(t, h.(prometheus.Metric).Write(&m))
	return m.GetHistogram()
}
