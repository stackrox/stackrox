package concurrency

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// RetryTicker repeatedly calls a function with a timeout and a retry backoff strategy.
type RetryTicker struct {
	f func(ctx context.Context) (timeToNextTick time.Duration, err error)
	tickTimeout time.Duration
	backoffPrototype wait.Backoff
	backoff      wait.Backoff
	tickTimer *time.Timer
	OnTickSuccess func(nextTimeToTick time.Duration)
	OnTickError func(error)
}

// NewRetryTicker returns a new RetryTicker that calls the function f repeatedly:
// - When started, the RetryTicker calls f immediately, and if it returns an error
// then the RetryTicker will wait the time returned by backoff.Step before calling f again.
// - f must return an error if ctx is cancelled. RetryTicker always call f with a context with a timeout of tickTimeout.
// - On success RetryTicker will reset backoff, and wait the amount of time returned by f before running f again.
func NewRetryTicker(f func(ctx context.Context) (nextTimeToTick time.Duration, err error),
	tickTimeout time.Duration,
	backoff wait.Backoff) *RetryTicker{
	ticker := &RetryTicker{
		f: f,
		tickTimeout: tickTimeout,
		backoffPrototype: backoff,
		backoff: backoff,
	}
	return ticker
}

func (t *RetryTicker) Start() {
	t.scheduleTick(0)
}

func (t *RetryTicker) Stop() {
	t.setTickTimer(nil)
}

func (t *RetryTicker) scheduleTick(timeToTick time.Duration) {
	t.setTickTimer(time.AfterFunc(timeToTick, func() {
		ctx, cancel := context.WithTimeout(context.Background(), t.tickTimeout)
		defer cancel()

		nextTimeToTick, tickErr := t.f(ctx)
		if tickErr != nil {
			if t.OnTickError != nil {
				t.OnTickError(tickErr)
			}
			t.scheduleTick(t.backoff.Step())
			return
		}
		if t.OnTickSuccess != nil {
			t.OnTickSuccess(nextTimeToTick)
		}
		t.backoff = t.backoffPrototype // reset backoff strategy
		t.scheduleTick(nextTimeToTick)
	}))
}

func (t *RetryTicker) setTickTimer(timer *time.Timer) {
	if t.tickTimer != nil {
		t.tickTimer.Stop()
	}
	t.tickTimer = timer
}
