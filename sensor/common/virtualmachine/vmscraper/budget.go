package vmscraper

import (
	"cmp"
	"hash/fnv"
	"slices"
	"time"

	"github.com/stackrox/rox/sensor/common/virtualmachine"
)

// tickStartBudget is the per-tick start cap from tracked fleet size, not the
// current due pile: leftovers must not shrink the rate. Catch-up if any
// never-forwarded VM is due, else the steady width; concurrency is a hard cap.
func tickStartBudget(nTracked, nUrgentDue, concurrency int, tick, catchUpWindow, steadyWidth time.Duration) int {
	if concurrency < 1 {
		concurrency = 1
	}
	if nTracked < 1 {
		return 0
	}
	window := steadyWidth
	if nUrgentDue > 0 {
		window = catchUpWindow
	}
	budget := startBudget(nTracked, tick, window)
	if budget < 1 {
		return 0
	}
	return min(budget, concurrency)
}

// budgetTickDuration is the time base for this tick's start budget: the
// nominal tick, or the wall-clock gap since the last tick if that was longer
// (a scrape Wait that overran the ticker).
func budgetTickDuration(nominal, elapsed time.Duration) time.Duration {
	return max(nominal, elapsed)
}

// startBudget is max(1, ceil(n × tick / window)) for positive n, tick, and
// window. Zero otherwise so a missing window does not start a scrape.
func startBudget(n int, tick, window time.Duration) int {
	if n <= 0 || tick <= 0 || window <= 0 {
		return 0
	}
	num := int64(n) * int64(tick)
	den := int64(window)
	b := int((num + den - 1) / den)
	if b < 1 {
		return 1
	}
	return b
}

type dueCandidate struct {
	key          string
	neverScraped bool
	hash         uint64
}

// selectDueStarts picks up to budget keys from the due set, preferring
// never-scraped VMs so first-wave work is not starved by cadenced peers.
func selectDueStarts(cands []dueCandidate, budget int) []string {
	if budget < 1 || len(cands) == 0 {
		return nil
	}
	sorted := slices.Clone(cands)
	slices.SortFunc(sorted, cmpDueCandidate)

	if budget > len(sorted) {
		budget = len(sorted)
	}
	out := make([]string, 0, budget)
	for i := range budget {
		out = append(out, sorted[i].key)
	}
	return out
}

func cmpDueCandidate(a, b dueCandidate) int {
	switch {
	case a.neverScraped && !b.neverScraped:
		return -1
	case !a.neverScraped && b.neverScraped:
		return 1
	}
	if c := cmp.Compare(a.hash, b.hash); c != 0 {
		return c
	}
	return cmp.Compare(a.key, b.key)
}

// hashVMID returns a stable 64-bit hash for ordering. Prefers VM ID; falls back
// to schedule key. Callers should cache the result (e.g. on vmState) so empty-ID
// fallback is not logged on every tick.
func hashVMID(id virtualmachine.VMID, key string) uint64 {
	s := string(id)
	if s == "" {
		log.Warnf("VMScraper: empty VM ID for schedule key %q; hashing namespace/name fallback", key)
		s = key
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}
