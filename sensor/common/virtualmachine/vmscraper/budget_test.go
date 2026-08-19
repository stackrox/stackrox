package vmscraper

import (
	"testing"
	"time"

	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stretchr/testify/assert"
)

func TestStartBudget(t *testing.T) {
	t.Parallel()

	tick := 10 * time.Second
	cases := map[string]struct {
		n      int
		tick   time.Duration
		window time.Duration
		want   int
	}{
		"n=100 catchUp=20m -> 1": {
			n: 100, tick: tick, window: 20 * time.Minute, want: 1,
		},
		"n=100 steadyWidth=40m -> 1": {
			n: 100, tick: tick, window: 40 * time.Minute, want: 1,
		},
		"n=100 window=5m -> ceil(100*10/300)=4": {
			n: 100, tick: tick, window: 5 * time.Minute, want: 4,
		},
		"n=0 -> 0": {
			n: 0, tick: tick, window: 20 * time.Minute, want: 0,
		},
		"zero window -> 0": {
			n: 10, tick: tick, window: 0, want: 0,
		},
		"zero tick -> 0": {
			n: 10, tick: 0, window: 20 * time.Minute, want: 0,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, startBudget(tc.n, tc.tick, tc.window))
		})
	}
}

func TestBudgetTickDuration(t *testing.T) {
	t.Parallel()
	tick := 10 * time.Second
	cases := map[string]struct {
		elapsed time.Duration
		want    time.Duration
	}{
		"first tick (no elapsed) uses nominal": {
			elapsed: 0, want: tick,
		},
		"on-time tick uses nominal": {
			elapsed: tick, want: tick,
		},
		"early tick still uses nominal": {
			elapsed: time.Second, want: tick,
		},
		"overrun uses elapsed": {
			elapsed: 30 * time.Second, want: 30 * time.Second,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, budgetTickDuration(tick, tc.elapsed))
		})
	}
}

func TestTickStartBudget(t *testing.T) {
	t.Parallel()

	tick := 10 * time.Second
	catchUp20m := 20 * time.Minute
	steady40m := 40 * time.Minute
	catchUp100s := 100 * time.Second
	steady200s := 200 * time.Second

	cases := map[string]struct {
		nTracked, nUrgentDue, concurrency int
		catchUp, steady                   time.Duration
		want                              int
	}{
		"100 tracked all urgent over 20m catch-up should yield 1 start": {
			nTracked: 100, nUrgentDue: 100, concurrency: 20,
			catchUp: catchUp20m, steady: steady40m, want: 1,
		},
		"100 tracked cadenced-only over 40m steady should yield 1 start": {
			nTracked: 100, nUrgentDue: 0, concurrency: 20,
			catchUp: catchUp20m, steady: steady40m, want: 1,
		},
		"cadenced pile with concurrency 1 should yield 1 start": {
			nTracked: 100, nUrgentDue: 0, concurrency: 1,
			catchUp: catchUp20m, steady: steady40m, want: 1,
		},
		"100 tracked urgent over a 5m catch-up window should yield 4 starts": {
			nTracked: 100, nUrgentDue: 100, concurrency: 20,
			catchUp: 5 * time.Minute, steady: steady40m, want: 4,
		},
		"one urgent due should use the fleet catch-up rate, not 1": {
			nTracked: 101, nUrgentDue: 1, concurrency: 20,
			catchUp: catchUp100s, steady: steady200s, want: 11,
		},
		"leftover due must not shrink a 100-VM catch-up rate": {
			nTracked: 100, nUrgentDue: 55, concurrency: 20,
			catchUp: catchUp100s, steady: steady200s, want: 10,
		},
		"zero tracked should yield no starts": {
			nTracked: 0, nUrgentDue: 0, concurrency: 20,
			catchUp: catchUp100s, steady: steady200s, want: 0,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tickStartBudget(
				tc.nTracked, tc.nUrgentDue, tc.concurrency, tick, tc.catchUp, tc.steady,
			))
		})
	}
}

func TestSelectDueStarts(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		cands     []dueCandidate
		budget    int
		wantOrder []string
	}{
		"preferring never-scraped then lower hash should yield new-a, new-z, cadenced-b": {
			cands: []dueCandidate{
				{key: "ns/cadenced-a", neverScraped: false, hash: 10},
				{key: "ns/new-z", neverScraped: true, hash: 90},
				{key: "ns/new-a", neverScraped: true, hash: 5},
				{key: "ns/cadenced-b", neverScraped: false, hash: 1},
			},
			budget:    3,
			wantOrder: []string{"ns/new-a", "ns/new-z", "ns/cadenced-b"},
		},
		"budget 0 should yield no starts": {
			cands: []dueCandidate{
				{key: "a", neverScraped: true, hash: 1},
				{key: "b", neverScraped: true, hash: 2},
				{key: "c", neverScraped: true, hash: 3},
			},
			budget:    0,
			wantOrder: nil,
		},
		"budget 1 among three never-scraped should yield only the lowest hash": {
			cands: []dueCandidate{
				{key: "a", neverScraped: true, hash: 1},
				{key: "b", neverScraped: true, hash: 2},
				{key: "c", neverScraped: true, hash: 3},
			},
			budget:    1,
			wantOrder: []string{"a"},
		},
		"budget above candidate count should yield every candidate in hash order": {
			cands: []dueCandidate{
				{key: "a", neverScraped: true, hash: 1},
				{key: "b", neverScraped: true, hash: 2},
				{key: "c", neverScraped: true, hash: 3},
			},
			budget:    10,
			wantOrder: []string{"a", "b", "c"},
		},
		"stable VM-ID hashes should yield the same order on every call": {
			cands: []dueCandidate{
				{key: "ns/vm-c", neverScraped: true, hash: hashVMID(virtualmachine.VMID("id-c"), "ns/vm-c")},
				{key: "ns/vm-a", neverScraped: true, hash: hashVMID(virtualmachine.VMID("id-a"), "ns/vm-a")},
				{key: "ns/vm-b", neverScraped: true, hash: hashVMID(virtualmachine.VMID("id-b"), "ns/vm-b")},
			},
			budget:    3,
			wantOrder: []string{"ns/vm-c", "ns/vm-b", "ns/vm-a"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := selectDueStarts(tc.cands, tc.budget)
			assert.Equal(t, tc.wantOrder, got)
			assert.Equal(t, got, selectDueStarts(tc.cands, tc.budget),
				"repeated selection should be deterministic")
		})
	}
}

func TestHashVMID_Stable(t *testing.T) {
	t.Parallel()
	a := hashVMID(virtualmachine.VMID("abc"), "ns/name")
	b := hashVMID(virtualmachine.VMID("abc"), "other/key")
	assert.Equal(t, a, b, "hash should key off VM ID when present")
	assert.NotEqual(t, hashVMID("", "ns/a"), hashVMID("", "ns/b"))
}
