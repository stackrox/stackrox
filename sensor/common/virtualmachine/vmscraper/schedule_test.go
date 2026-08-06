package vmscraper

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
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

func TestIsRetryable(t *testing.T) {
	assert.True(t, isRetryable(io.EOF))
	assert.True(t, isRetryable(vsockclient.ErrNotReady))
	assert.True(t, isRetryable(vsockclient.ErrInternal))
	assert.True(t, isRetryable(errors.New("dial failed")))
	assert.False(t, isRetryable(vsockclient.ErrUnknownMethod))
	assert.False(t, isRetryable(nil))
}
