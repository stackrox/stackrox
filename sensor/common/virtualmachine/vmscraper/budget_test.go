package vmscraper

import (
	"testing"
	"time"

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
		"n=1 catchUp=20m -> max(1, ceil)": {
			n: 1, tick: tick, window: 20 * time.Minute, want: 1,
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
