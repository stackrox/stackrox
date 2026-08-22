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

	// catchUpBound caps the first-wave spread window so a Sensor restart
	// does not take forever to reach every VM at long poll intervals.
	catchUpBound = 20 * time.Minute

	// defaultSpreadFraction is the fraction of pollInterval used as a
	// one-sided random band when returning to cadence: next = now +
	// poll + U(0, fraction*poll). Prevents VMs from re-aligning on
	// the same wall-clock minute after each successful scrape.
	defaultSpreadFraction = 2.0 / 3
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

// catchUpWindow is how long the first wave of never-scraped VMs is spread
// over: min(catchUpBound, pollInterval/3).
func catchUpWindow(pollInterval time.Duration) time.Duration {
	if pollInterval <= 0 {
		return 0
	}
	return min(catchUpBound, pollInterval/3)
}

// steadySpreadWidth returns the post-poll random band width:
// spreadFraction × pollInterval.
func steadySpreadWidth(pollInterval time.Duration) time.Duration {
	if pollInterval <= 0 {
		return 0
	}
	return time.Duration(float64(pollInterval) * defaultSpreadFraction)
}

// randOffset draws a delay in [0, width] from a unit sample in [0, 1].
func randOffset(width time.Duration, unit float64) time.Duration {
	if width <= 0 {
		return 0
	}
	unit = max(0, min(1, unit))
	return time.Duration(float64(width) * unit)
}
