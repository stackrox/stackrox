package gocache

import (
	"sort"
	"sync"
	"time"
)

// TimeoutTracker adapts the cloud-GET timeout to observed latency so a
// slow or hung blob store degrades to recompiles instead of stalling
// builds (go recompiles on miss and re-puts — self-healing). Ported,
// simplified, from gocache-serve: P95 of recent successful fetches
// with 4× headroom, floored and capped.
type TimeoutTracker struct {
	mu     sync.Mutex
	window []time.Duration
	max    int
	def    time.Duration
}

func NewTimeoutTracker() *TimeoutTracker {
	return &TimeoutTracker{max: 64, def: 3 * time.Second}
}

// Timeout returns the current per-request deadline.
func (t *TimeoutTracker) Timeout() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.window) == 0 {
		return t.def
	}
	s := append([]time.Duration{}, t.window...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	p95 := s[len(s)*95/100]
	d := p95 * 4 // headroom over observed latency
	if d < 500*time.Millisecond {
		d = 500 * time.Millisecond // floor the RESULT: fast stores get snappy timeouts
	}
	if d > 15*time.Second {
		d = 15 * time.Second
	}
	return d
}

// Record notes a successful fetch duration.
func (t *TimeoutTracker) Record(d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.window = append(t.window, d)
	if len(t.window) > t.max {
		t.window = t.window[len(t.window)-t.max:]
	}
}
