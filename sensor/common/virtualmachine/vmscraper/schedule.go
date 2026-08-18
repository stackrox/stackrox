package vmscraper

import (
	"time"
)

const (
	initialBackoff = 10 * time.Second
	maxBackoffCap  = 30 * time.Minute
	maxReconcile   = 5 * time.Minute
	// defaultTickInterval is the scraper scheduler step. Independent of retry-backoff.
	defaultTickInterval = 10 * time.Second
)

func backoffCap(pollInterval time.Duration) time.Duration {
	return min(pollInterval, maxBackoffCap)
}

func reconcilePeriod(pollInterval time.Duration) time.Duration {
	return min(pollInterval, maxReconcile)
}

// nextBackoff returns the next backoff after a retryable failure or NACK.
func nextBackoff(current, pollInterval, initial time.Duration) time.Duration {
	limit := backoffCap(pollInterval)
	if current <= 0 {
		return min(initial, limit)
	}
	return min(current*2, limit)
}
