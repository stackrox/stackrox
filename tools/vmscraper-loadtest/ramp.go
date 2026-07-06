package main

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vmscraper/loadtest"
)

// backlogGauge is harness-only (not a real production vsock_pull_* metric,
// hence the distinct "loadtest_" name) -- see design spec § Observability,
// "Backlog gauge". Sampled once per poll interval by runRamp, and also read
// once for the end-of-run summary in non-ramp mode.
var backlogGauge = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: "rox",
	Subsystem: "sensor",
	Name:      "vsock_pull_loadtest_backlog_vms",
	Help:      "Harness-only: number of synthetic VMs whose time since last successful scrape exceeds the configured poll interval.",
})

func init() {
	prometheus.MustRegister(backlogGauge)
}

// rampConfig configures ramp mode's step size/cadence and safety limits.
type rampConfig struct {
	step       int
	stepCycles int // K: both the hold time (in poll cycles) between steps, and the trend window for breach detection
	maxVMs     int // safety cap, 0 = unlimited
}

// rampResult reports how runRamp ended.
type rampResult struct {
	lastStableVMs int
	finalVMs      int
	breached      bool
}

// runRamp implements the design spec's ramp mode: starting from farm's
// current size, step the fleet up by cfg.step every cfg.stepCycles poll
// intervals until the backlog gauge shows a sustained growing queue (K
// consecutive non-zero, strictly increasing samples), or a safety limit
// (cfg.maxVMs, or ctx being done -- e.g. --duration elapsed) is hit.
//
// This directly automates the search for "how many VMs can a single Sensor
// support" (Q1 in the design spec) instead of manual trial-and-error runs.
func runRamp(ctx context.Context, farm *loadtest.Farm, pollInterval time.Duration, cfg rampConfig) rampResult {
	current := farm.Count()
	lastStable := current
	cyclesAtStep := 0
	history := make([]int, 0, cfg.stepCycles)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Infof("vmscraper-loadtest: ramp: stopping (duration elapsed or interrupted) at %d VMs, no breach observed", current)
			return rampResult{lastStableVMs: lastStable, finalVMs: current, breached: false}
		case <-ticker.C:
			backlog := farm.BacklogCount(pollInterval)
			backlogGauge.Set(float64(backlog))
			history = append(history, backlog)
			if len(history) > cfg.stepCycles {
				history = history[1:]
			}
			log.Infof("vmscraper-loadtest: ramp: %d VMs, backlog=%d (history=%v)", current, backlog, history)

			if isSustainedGrowth(history, cfg.stepCycles) {
				log.Warnf("vmscraper-loadtest: ramp: sustained backlog growth detected at %d VMs (last stable size: %d)", current, lastStable)
				return rampResult{lastStableVMs: lastStable, finalVMs: current, breached: true}
			}

			cyclesAtStep++
			if cyclesAtStep < cfg.stepCycles {
				continue
			}
			lastStable = current
			if cfg.maxVMs > 0 && current+cfg.step > cfg.maxVMs {
				log.Infof("vmscraper-loadtest: ramp: reached --ramp-max-vms=%d without a backlog breach", cfg.maxVMs)
				return rampResult{lastStableVMs: lastStable, finalVMs: current, breached: false}
			}
			farm.AddVMs(cfg.step)
			current += cfg.step
			cyclesAtStep = 0
			// Growing the farm causes a transient backlog bump (new VMs all
			// start "due" at once) that would otherwise look identical to a
			// real breach for the next sample or two. Reset the trend window
			// so only post-step-up behavior counts toward breach detection.
			history = history[:0]
			log.Infof("vmscraper-loadtest: ramp: stepped up to %d VMs", current)
		}
	}
}

// isSustainedGrowth reports whether history holds k samples and is strictly
// increasing throughout, with no zero entries -- the design spec's breach
// criterion: "the backlog gauge is non-zero and increasing across K
// consecutive cycles", not just momentarily non-empty.
func isSustainedGrowth(history []int, k int) bool {
	if len(history) < k || history[0] == 0 {
		return false
	}
	for i := 1; i < len(history); i++ {
		if history[i] <= history[i-1] {
			return false
		}
	}
	return true
}
