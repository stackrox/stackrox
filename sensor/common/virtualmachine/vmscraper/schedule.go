package vmscraper

import (
	"math"
	"time"
)

const (
	initialBackoff = 10 * time.Second
	maxBackoffCap  = 30 * time.Minute
	maxReconcile   = 5 * time.Minute
	// defaultTickInterval is the scraper scheduler step. Independent of retry-backoff.
	defaultTickInterval = 10 * time.Second
	// maxNewVMIndexReportWindow caps the schedule window for a newly added VM's
	// first index report when the poll interval is long.
	maxNewVMIndexReportWindow         = 20 * time.Minute
	newVMIndexReportWindowPollDivisor = 3
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

// newVMIndexReportWindow is the schedule window for a newly added VM's first
// index report (min(maxNewVMIndexReportWindow, poll/3)).
func newVMIndexReportWindow(pollInterval time.Duration) time.Duration {
	if pollInterval <= 0 {
		return 0
	}
	return min(maxNewVMIndexReportWindow, pollInterval/newVMIndexReportWindowPollDivisor)
}

// suggestedPollInterval is the poll interval whose steady-state spread width
// fits numVMs at roughly one VM per tick.
func suggestedPollInterval(numVMs int, tick time.Duration, spreadFraction float64) time.Duration {
	if numVMs <= 0 || tick <= 0 || spreadFraction <= 0 {
		return 0
	}
	want := time.Duration(math.Ceil(float64(numVMs) * float64(tick) / spreadFraction))
	return max(minPollInterval, want)
}

// maxVMsForSteadyState is how many VMs fit in the steady-state spread window
// at one per tick.
func maxVMsForSteadyState(tick, poll time.Duration, spreadFraction float64) int {
	if tick <= 0 || spreadFraction <= 0 {
		return 0
	}
	return int(steadySpreadWidth(poll, spreadFraction) / tick)
}

// maxVMsForNewVMIndexReportWindow is how many VMs fit in the new-VM index
// report window at one per tick. Extra VMs share ticks; this is density, not a hard limit.
func maxVMsForNewVMIndexReportWindow(tick, poll time.Duration) int {
	if tick <= 0 {
		return 0
	}
	return int(newVMIndexReportWindow(poll) / tick)
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
