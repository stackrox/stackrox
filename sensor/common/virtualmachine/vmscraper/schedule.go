package vmscraper

import (
	"time"
)

const (
	initialBackoff = 10 * time.Second
	maxBackoffCap  = 30 * time.Minute
	maxReconcile   = 5 * time.Minute
	// defaultTickInterval is the scraper scheduler step and the time base for
	// the per-tick start budget. Independent of retry backoff so it can change
	// without retuning NACK/failure delays.
	defaultTickInterval = 10 * time.Second
	// catchUpBound is the upper operand in catchUpWindow = min(bound, pollInterval/3).
	catchUpBound = 20 * time.Minute
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

// steadySpreadWidth is how wide the post-poll random band is when a VM
// returns to normal cadence (spreadFraction × pollInterval).
func steadySpreadWidth(pollInterval time.Duration, spreadFraction float64) time.Duration {
	if pollInterval <= 0 || spreadFraction <= 0 {
		return 0
	}
	return time.Duration(float64(pollInterval) * spreadFraction)
}

// catchUpWindow is how long a mass first-wave of never-scraped VMs may be
// spread over (min(20m, pollInterval/3)), not the steady band.
func catchUpWindow(pollInterval time.Duration) time.Duration {
	if pollInterval <= 0 {
		return 0
	}
	return min(catchUpBound, pollInterval/3)
}

// randOffset draws a delay in [0, max] from a unit sample in [0, 1].
func randOffset(max time.Duration, unit float64) time.Duration {
	if max <= 0 {
		return 0
	}
	if unit < 0 {
		unit = 0
	}
	if unit > 1 {
		unit = 1
	}
	return time.Duration(float64(max) * unit)
}
