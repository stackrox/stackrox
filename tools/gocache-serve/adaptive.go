package main

import (
	"log"
	"sync"
	"time"
)

// AdaptiveSem is a concurrency limiter that self-tunes based on observed
// request latency, with FIFO priority: the oldest waiting request always
// gets the next available slot. It implements AIMD (Additive Increase,
// Multiplicative Decrease) — the same principle as TCP congestion control:
//   - Start conservative (min concurrent requests)
//   - If latency is stable, increase by 1 (probe for more bandwidth)
//   - If latency degrades by >50%, halve (back off from congestion)
//
// FIFO priority matters because Go's compilation has a critical path:
// the oldest pending cache entry is likely blocking a compilation step
// that other packages depend on. Prioritizing it minimizes wall-clock.
type AdaptiveSem struct {
	mu       sync.Mutex
	limit    int
	min      int
	max      int
	inflight int

	waiters []chan struct{} // FIFO queue: oldest waiter gets next slot

	latencies   []time.Duration
	windowSize  int
	adjustCount int
}

func NewAdaptiveSem(min, max int) *AdaptiveSem {
	return &AdaptiveSem{
		limit:      min,
		min:        min,
		max:        max,
		windowSize: 10,
	}
}

// Acquire blocks until a concurrency slot is available.
// Waiters are served in FIFO order — the oldest request goes first.
func (as *AdaptiveSem) Acquire() {
	as.mu.Lock()

	// Fast path: slot available and no older waiters
	if as.inflight < as.limit && len(as.waiters) == 0 {
		as.inflight++
		as.mu.Unlock()
		return
	}

	// Queue ourselves (FIFO — appended to the end)
	ready := make(chan struct{})
	as.waiters = append(as.waiters, ready)
	as.mu.Unlock()

	// Block until Release hands us a slot
	<-ready
}

// Release returns a slot, records the latency for tuning, and immediately
// admits the oldest waiter (if any). When the adaptive limit increases,
// additional waiters are admitted right away.
func (as *AdaptiveSem) Release(latency time.Duration) {
	as.mu.Lock()
	as.inflight--

	// Wake the oldest waiter
	as.wakeWaiters()

	as.mu.Unlock()

	// Record latency and adjust (called outside lock to avoid deadlock
	// if adjust() logs and the log callback needs the mutex)
	as.mu.Lock()
	as.recordLatency(latency)
	as.mu.Unlock()
}

// wakeWaiters admits as many waiters as the current limit allows.
// Must be called with as.mu held.
func (as *AdaptiveSem) wakeWaiters() {
	for as.inflight < as.limit && len(as.waiters) > 0 {
		ready := as.waiters[0]
		as.waiters = as.waiters[1:]
		as.inflight++
		close(ready)
	}
}

func (as *AdaptiveSem) Limit() int {
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.limit
}

// recordLatency adds a sample and adjusts the limit when enough data exists.
// Must be called with as.mu held.
func (as *AdaptiveSem) recordLatency(latency time.Duration) {
	as.latencies = append(as.latencies, latency)
	if len(as.latencies) > as.windowSize*2 {
		as.latencies = as.latencies[1:]
	}

	as.adjustCount++
	if as.adjustCount >= as.windowSize {
		as.adjustCount = 0
		before := as.limit
		as.adjust()
		// If the limit increased, admit more waiters immediately
		if as.limit > before {
			as.wakeWaiters()
		}
	}
}

// adjust examines recent latency trends and adjusts the limit.
// Must be called with as.mu held.
func (as *AdaptiveSem) adjust() {
	n := len(as.latencies)
	if n < as.windowSize {
		return
	}

	mid := n / 2
	var recentSum, olderSum time.Duration
	for i, d := range as.latencies {
		if i < mid {
			olderSum += d
		} else {
			recentSum += d
		}
	}
	recent := recentSum / time.Duration(n-mid)
	older := olderSum / time.Duration(mid)

	if older == 0 {
		return
	}

	ratio := float64(recent) / float64(older)

	switch {
	case ratio > 1.5:
		// Latency degraded significantly — congestion or rate limiting.
		// Back off aggressively (multiplicative decrease).
		newLimit := as.limit / 2
		if newLimit < as.min {
			newLimit = as.min
		}
		if newLimit != as.limit {
			log.Printf("gocache-serve: concurrency %d -> %d (latency %.1fx worse: %.0fms -> %.0fms)",
				as.limit, newLimit, ratio, float64(older)/1e6, float64(recent)/1e6)
			as.limit = newLimit
		}

	case ratio < 1.2:
		// Latency stable or improving — bandwidth available.
		// Probe with one more concurrent request (additive increase).
		if as.limit < as.max {
			as.limit++
			log.Printf("gocache-serve: concurrency %d -> %d (latency stable at %.0fms)",
				as.limit-1, as.limit, float64(recent)/1e6)
		}
	}
}
