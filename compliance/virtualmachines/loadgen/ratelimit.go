package main

import (
	"sync"
	"sync/atomic"
	"time"
)

// errorLogLimiter rate-limits error logging to prevent log spam.
type errorLogLimiter struct {
	mu           sync.Mutex
	lastLogTime  time.Time
	minInterval  time.Duration
	droppedCount atomic.Uint64
}

func newErrorLogLimiter(minInterval time.Duration) *errorLogLimiter {
	return &errorLogLimiter{
		minInterval: minInterval,
	}
}

func (e *errorLogLimiter) shouldLog() (bool, uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	if now.Sub(e.lastLogTime) >= e.minInterval {
		e.lastLogTime = now
		dropped := e.droppedCount.Swap(0)
		return true, dropped
	}
	e.droppedCount.Add(1)
	return false, 0
}
