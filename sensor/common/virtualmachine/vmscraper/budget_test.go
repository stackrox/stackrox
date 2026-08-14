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
		"n=0 -> 1": {
			n: 0, tick: tick, window: 20 * time.Minute, want: 1,
		},
		"zero window -> 1": {
			n: 10, tick: tick, window: 0, want: 1,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, startBudget(tc.n, tc.tick, tc.window))
		})
	}
}

func TestTickStartBudget(t *testing.T) {
	t.Parallel()

	tick := 10 * time.Second
	catchUp := 20 * time.Minute
	steadyWidth := 40 * time.Minute

	assert.Equal(t, 1, tickStartBudget(100, 100, 20, tick, catchUp, steadyWidth),
		"urgent uses catch-up budget, not full concurrency")
	assert.Equal(t, 1, tickStartBudget(0, 100, 20, tick, catchUp, steadyWidth),
		"cadenced uses steady width budget capped by concurrency")
	assert.Equal(t, 1, tickStartBudget(0, 100, 1, tick, catchUp, steadyWidth))
	assert.Equal(t, 4, tickStartBudget(100, 100, 20, tick, 5*time.Minute, steadyWidth))
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
