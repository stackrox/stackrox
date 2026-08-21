package vmscraper

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
}

func TestVMScraper_ManyCadencedSuccessesSpanSteadyBand(t *testing.T) {
	const numVMs = 5
	vms := make([]*virtualmachine.Info, 0, numVMs)
	for i := range numVMs {
		vms = append(vms, makeVM("ns", fmt.Sprintf("vm-%d", i), uint32(10+i)))
	}
	store := &mockStore{vms: vms}
	s, clock := newTestScraper(store, &safeSender{}, &mockDialer{}, &safeProtocolClient{gen: 1})
	s.concurrency = numVMs
	s.interval = time.Hour
	s.spreadFraction = 2.0 / 3
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

func TestVMScraper_MassFirstInsertSpansCatchUpWindow(t *testing.T) {
	const numVMs = 10
	vms := make([]*virtualmachine.Info, 0, numVMs)
	for i := range numVMs {
		vms = append(vms, makeVM("ns", fmt.Sprintf("vm-%d", i), uint32(100+i)))
	}
	store := &mockStore{vms: vms}
	s, clock := newTestScraper(store, &mockSender{}, &mockDialer{}, &safeProtocolClient{gen: 1})
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
	s, clock := newTestScraper(store, &mockSender{}, &mockDialer{}, &safeProtocolClient{gen: 1})
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
	s, clock := newTestScraper(store, &mockSender{}, &mockDialer{}, &safeProtocolClient{gen: 1})
	s.concurrency = 20
	s.randFloat64 = sequencedRand([]float64{0.25, 0.75})

	s.reconcile()

	catchUp := catchUpWindow(s.interval)
	assert.Equal(t, clock.Now().Add(catchUp/4), cachedNextAttemptAt(t, s, "ns/a"))
	assert.Equal(t, clock.Now().Add(3*catchUp/4), cachedNextAttemptAt(t, s, "ns/b"))
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

func TestSpreadFractionEnvDefault(t *testing.T) {
	assert.InDelta(t, 2.0/3, env.VirtualMachinesScraperSteadySpreadFraction.DefaultValue(), 1e-9)
	assert.InDelta(t, 2.0/3, env.VirtualMachinesScraperSteadySpreadFraction.FloatSetting(), 1e-9)
}
