package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

// TimeoutTracker computes an adaptive timeout for cloud GETs based on the
// observed latency distribution. The insight: a slow response is worse than
// a miss — a miss triggers recompilation (known cost), while a slow response
// blocks Go's critical path for an unknown duration. Returning a Miss on
// timeout lets Go recompile, and the result gets `put` back to the cache
// (self-healing).
//
// Adaptive: timeout = P95(recent latencies) × multiplier, clamped to [min, max].
// This self-tunes for same-region (~50ms normal, ~150ms timeout) vs
// cross-region (~250ms normal, ~750ms timeout) vs degraded network.
//
// Configuration (env vars):
//
//	GOCACHE_SERVE_TIMEOUT      fixed timeout in ms (0 = adaptive)
//	GOCACHE_SERVE_TIMEOUT_MULT adaptive multiplier (default 3.0)
//	GOCACHE_SERVE_TIMEOUT_MIN  minimum timeout in ms (default 100)
//	GOCACHE_SERVE_TIMEOUT_MAX  maximum timeout in ms (default 5000)
type TimeoutTracker struct {
	mu        sync.Mutex
	latencies []time.Duration

	fixed      time.Duration // if > 0, use this instead of adaptive
	multiplier float64
	minimum    time.Duration
	maximum    time.Duration
}

func NewTimeoutTracker() *TimeoutTracker {
	tt := &TimeoutTracker{
		multiplier: 3.0,
		minimum:    100 * time.Millisecond,
		maximum:    5 * time.Second,
	}

	// Parse env vars
	if v := envMs("GOCACHE_SERVE_TIMEOUT"); v > 0 {
		tt.fixed = time.Duration(v) * time.Millisecond
	}
	if v := envFloat("GOCACHE_SERVE_TIMEOUT_MULT"); v > 0 {
		tt.multiplier = v
	}
	if v := envMs("GOCACHE_SERVE_TIMEOUT_MIN"); v > 0 {
		tt.minimum = time.Duration(v) * time.Millisecond
	}
	if v := envMs("GOCACHE_SERVE_TIMEOUT_MAX"); v > 0 {
		tt.maximum = time.Duration(v) * time.Millisecond
	}

	return tt
}

// Timeout returns the current timeout for a cloud GET.
func (tt *TimeoutTracker) Timeout() time.Duration {
	if tt.fixed > 0 {
		return tt.fixed
	}

	tt.mu.Lock()
	defer tt.mu.Unlock()

	if len(tt.latencies) < 5 {
		return tt.maximum // not enough data, be generous
	}

	// P95 of recent latencies
	sorted := make([]time.Duration, len(tt.latencies))
	copy(sorted, tt.latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	p95Idx := int(float64(len(sorted)) * 0.95)
	if p95Idx >= len(sorted) {
		p95Idx = len(sorted) - 1
	}
	p95 := sorted[p95Idx]

	timeout := time.Duration(float64(p95) * tt.multiplier)
	if timeout < tt.minimum {
		timeout = tt.minimum
	}
	if timeout > tt.maximum {
		timeout = tt.maximum
	}
	return timeout
}

// Record adds a latency sample.
func (tt *TimeoutTracker) Record(latency time.Duration, timedOut bool) {
	// Don't record timed-out requests — they'd skew the P95 upward,
	// which would increase the timeout (wrong direction).
	if timedOut {
		return
	}

	tt.mu.Lock()
	defer tt.mu.Unlock()

	tt.latencies = append(tt.latencies, latency)
	if len(tt.latencies) > 50 {
		tt.latencies = tt.latencies[1:]
	}
}

// Stats returns current timeout state for logging.
func (tt *TimeoutTracker) Stats() string {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	if tt.fixed > 0 {
		return fmt.Sprintf("fixed %v", tt.fixed)
	}
	if len(tt.latencies) < 5 {
		return fmt.Sprintf("adaptive (warming up, max %v)", tt.maximum)
	}
	return fmt.Sprintf("adaptive %v (P95-based, %d samples)", tt.Timeout(), len(tt.latencies))
}

func envMs(key string) int {
	v := os.Getenv(key)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func envFloat(key string) float64 {
	v := os.Getenv(key)
	if v == "" {
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f <= 0 {
		return 0
	}
	return f
}
