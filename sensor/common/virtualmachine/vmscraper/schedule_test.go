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
		want    time.Duration
	}{
		"first failure uses initial backoff": {
			current: 0,
			poll:    poll,
			want:    initialBackoff,
		},
		"doubles until cap": {
			current: initialBackoff,
			poll:    poll,
			want:    20 * time.Second,
		},
		"caps at poll interval when poll is below maxBackoffCap": {
			current: 4 * time.Minute,
			poll:    poll,
			want:    poll,
		},
		"caps at maxBackoffCap when poll is larger": {
			current: 20 * time.Minute,
			poll:    time.Hour,
			want:    maxBackoffCap,
		},
		"first failure is capped when poll is below initial backoff": {
			current: 0,
			poll:    5 * time.Second,
			want:    5 * time.Second,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, nextBackoff(tc.current, tc.poll))
		})
	}
}

func TestReconcilePeriod(t *testing.T) {
	assert.Equal(t, 5*time.Minute, reconcilePeriod(5*time.Minute))
	assert.Equal(t, 5*time.Minute, reconcilePeriod(time.Hour))
	assert.Equal(t, time.Minute, reconcilePeriod(time.Minute))
}
