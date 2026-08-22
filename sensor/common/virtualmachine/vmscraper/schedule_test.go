package vmscraper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNextBackoff(t *testing.T) {
	poll := 5 * time.Minute
	cases := map[string]struct {
		current time.Duration
		poll    time.Duration
		initial time.Duration
		want    time.Duration
	}{
		"first failure uses initial backoff": {
			current: 0,
			poll:    poll,
			initial: initialBackoff,
			want:    initialBackoff,
		},
		"first failure uses the provided initial backoff": {
			current: 0,
			poll:    poll,
			initial: 30 * time.Second,
			want:    30 * time.Second,
		},
		"doubles until cap": {
			current: initialBackoff,
			poll:    poll,
			initial: initialBackoff,
			want:    20 * time.Second,
		},
		"caps at poll interval when poll is below maxBackoffCap": {
			current: 4 * time.Minute,
			poll:    poll,
			initial: initialBackoff,
			want:    poll,
		},
		"caps at maxBackoffCap when poll is larger": {
			current: 20 * time.Minute,
			poll:    time.Hour,
			initial: initialBackoff,
			want:    maxBackoffCap,
		},
		"first failure is capped when poll is below initial backoff": {
			current: 0,
			poll:    5 * time.Second,
			initial: initialBackoff,
			want:    5 * time.Second,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, nextBackoff(tc.current, tc.poll, tc.initial))
		})
	}
}

func TestReconcilePeriod(t *testing.T) {
	assert.Equal(t, 5*time.Minute, reconcilePeriod(5*time.Minute))
	assert.Equal(t, 5*time.Minute, reconcilePeriod(time.Hour))
	assert.Equal(t, time.Minute, reconcilePeriod(time.Minute))
}

func TestCatchUpWindow(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 20*time.Minute, catchUpWindow(time.Hour))
	assert.Equal(t, 100*time.Second, catchUpWindow(5*time.Minute))
	assert.Equal(t, time.Duration(0), catchUpWindow(0))
}

func TestSteadySpreadWidth(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 40*time.Minute, steadySpreadWidth(time.Hour))
	assert.Equal(t, time.Duration(0), steadySpreadWidth(0))
}

func TestRandOffset(t *testing.T) {
	t.Parallel()
	assert.Equal(t, time.Duration(0), randOffset(0, 0.5))
	assert.Equal(t, 20*time.Minute, randOffset(40*time.Minute, 0.5))
	assert.Equal(t, time.Duration(0), randOffset(40*time.Minute, 0))
	assert.Equal(t, 40*time.Minute, randOffset(40*time.Minute, 1))
	assert.Equal(t, time.Duration(0), randOffset(40*time.Minute, -1))
}
