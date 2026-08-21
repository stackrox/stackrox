package vmscraper

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sequencedRand returns successive values from seq, cycling the last value.
// Not goroutine-safe; only use with concurrency=1 (the newTestScraper default).
func sequencedRand(seq ...float64) func() float64 {
	i := 0
	return func() float64 {
		v := seq[i]
		if i < len(seq)-1 {
			i++
		}
		return v
	}
}

func TestVMScraper_ReconcileSpreadsCatchUpWindow(t *testing.T) {
	const numVMs = 10
	vms := make([]*virtualmachine.Info, numVMs)
	for i := range vms {
		vms[i] = makeVM("ns", fmt.Sprintf("vm-%d", i), uint32(100+i))
	}
	store := &mockStore{vms: vms}
	s, clock := newTestScraper(store, &mockSender{}, &mockDialer{}, &mockProtocolClient{})
	s.interval = time.Hour
	// Evenly spaced draws: 0/9, 1/9, ..., 9/9
	units := make([]float64, numVMs)
	for i := range units {
		units[i] = float64(i) / float64(numVMs-1)
	}
	s.randFloat64 = sequencedRand(units...)

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
	assert.GreaterOrEqual(t, latest.Sub(earliest), catchUp/2,
		"draws should span at least half the catch-up window")
}

func TestVMScraper_CadenceRescheduleUsesSteadyBand(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{makeVM("ns", "vm-a", 1)}}
	s, clock := newTestScraper(store, &mockSender{}, &mockDialer{}, &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport(1)},
	})
	// First draw (reconcile): 0 → due now. Second draw (cadence): 0.5.
	s.randFloat64 = sequencedRand(0, 0.5)

	start := clock.Now()
	s.pollOnce(context.Background())

	steadyWidth := steadySpreadWidth(s.interval)
	want := start.Add(s.interval + steadyWidth/2)
	assert.Equal(t, want, cachedNextAttemptAt(t, s, "ns/vm-a"),
		"cadence reschedule should use interval + random offset in steady band")
}

func TestVMScraper_RetryDoesNotUseSteadyBand(t *testing.T) {
	store := &mockStore{vms: []*virtualmachine.Info{makeVM("ns1", "vm-a", 100)}}
	client := &mockProtocolClient{errQueue: []error{vsockclient.ErrNotReady}}
	s, clock := newTestScraper(store, &mockSender{}, &mockDialer{}, client)
	// Reconcile draw: 0 → due now. Cadence draw if reached: 1 → full width.
	s.randFloat64 = sequencedRand(0, 1)

	start := clock.Now()
	s.pollOnce(context.Background())
	assert.Equal(t, initialBackoff, cachedNextAttemptAt(t, s, "ns1/vm-a").Sub(start),
		"retryable path must use backoff, not interval+offset")
}

func TestVMScraper_NACKDoesNotUseSteadyBand(t *testing.T) {
	vm := makeVM("ns1", "vm-a", 100)
	store := &mockStore{vms: []*virtualmachine.Info{vm}}
	s, clock := newTestScraper(store, &mockSender{}, &mockDialer{}, &mockProtocolClient{
		resultQueue: []*vsockclient.GetReportResult{makeReport(1)},
	})
	s.randFloat64 = sequencedRand(0, 1)
	s.pollOnce(context.Background())

	start := clock.Now()
	s.handleNACK(string(vm.ID) + ":1")
	assert.Equal(t, initialBackoff, cachedNextAttemptAt(t, s, "ns1/vm-a").Sub(start),
		"NACK must use backoff, not interval+offset")
}
