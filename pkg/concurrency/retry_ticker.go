package concurrency

import (
	"context"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	_ RetryTicker        = (*retryTickerImpl)(nil)
	_ RetryTickerBuilder = (*retryTickerBuilderImpl)(nil)
)

// RetryTicker repeatedly calls a function with a timeout and a retry backoff strategy.
type RetryTicker interface {
	Start()
	Stop()
}

type retryTickerImpl struct {
	f                tickFunc
	tickTimeout      time.Duration
	backoffPrototype wait.Backoff
	onTickSuccess    onTickSuccessFunc
	onTickError      onTickErrorFunc
	backoff          wait.Backoff
	tickTimer        *time.Timer
	tickTimerM       sync.Mutex
}

type tickFunc func(ctx context.Context) (timeToNextTick time.Duration, err error)
type onTickSuccessFunc func(nextTimeToTick time.Duration)
type onTickErrorFunc func(tickErr error)

// Start calls t.f and schedules the next tick accordingly.
func (t *retryTickerImpl) Start() {
	t.scheduleTick(0)
}

// Stop cancels this RetryTicker.
func (t *retryTickerImpl) Stop() {
	t.setTickTimer(nil)
}

func (t *retryTickerImpl) scheduleTick(timeToTick time.Duration) {
	t.setTickTimer(time.AfterFunc(timeToTick, func() {
		ctx, cancel := context.WithTimeout(context.Background(), t.tickTimeout)
		defer cancel()

		nextTimeToTick, tickErr := t.f(ctx)
		if tickErr != nil {
			if t.onTickError != nil {
				t.onTickError(tickErr)
			}
			t.scheduleTick(t.backoff.Step())
			return
		}
		if t.onTickSuccess != nil {
			t.onTickSuccess(nextTimeToTick)
		}
		t.backoff = t.backoffPrototype // reset backoff strategy
		t.scheduleTick(nextTimeToTick)
	}))
}

func (t *retryTickerImpl) setTickTimer(timer *time.Timer) {
	t.tickTimerM.Lock()
	defer t.tickTimerM.Unlock()
	if t.tickTimer != nil {
		t.tickTimer.Stop()
	}
	t.tickTimer = timer
}

// RetryTickerBuilder is a builder for RetryTicker objects.
type RetryTickerBuilder interface {
	OnTickSuccess(onTickSuccessFunc) RetryTickerBuilder
	OnTickError(onTickErrorFunc) RetryTickerBuilder
	Build() RetryTicker
}

// NewRetryTicker returns a new RetryTicker with the minimal parameters. See Build method below for
// details about how that is created.
func NewRetryTicker(f tickFunc, tickTimeout time.Duration, backoff wait.Backoff) RetryTicker {
	return NewRetryTickerBuilder(f, tickTimeout, backoff).Build()
}

// NewRetryTickerBuilder returns a builder for a RetryTicker that has been initialized with its mandatory parameters.
func NewRetryTickerBuilder(f tickFunc, tickTimeout time.Duration, backoff wait.Backoff) RetryTickerBuilder {
	return &retryTickerBuilderImpl{f: f, tickTimeout: tickTimeout, backoffPrototype: backoff}
}

// Build returns a new RetryTicker that calls the function f repeatedly:
// - When started, the RetryTicker calls f immediately, and if that returns an error
// then the RetryTicker will wait the time returned by backoff.Step before calling f again.
// - f must return an error if ctx is cancelled. RetryTicker always call f with a context with a timeout of tickTimeout.
// - On success RetryTicker will reset backoff, and wait the amount of time returned by f before running f again.
func (b *retryTickerBuilderImpl) Build() RetryTicker {
	return &retryTickerImpl{
		f:                b.f,
		tickTimeout:      b.tickTimeout,
		backoffPrototype: b.backoffPrototype,
		onTickSuccess:    b.onTickSuccess,
		onTickError:      b.onTickError,
		backoff:          b.backoffPrototype,
	}
}

type retryTickerBuilderImpl struct {
	f                tickFunc
	tickTimeout      time.Duration
	backoffPrototype wait.Backoff
	onTickSuccess    onTickSuccessFunc
	onTickError      onTickErrorFunc
}

func (b *retryTickerBuilderImpl) OnTickSuccess(onTickSuccess onTickSuccessFunc) RetryTickerBuilder {
	b.onTickSuccess = onTickSuccess
	return b
}

func (b *retryTickerBuilderImpl) OnTickError(onTickError onTickErrorFunc) RetryTickerBuilder {
	b.onTickError = onTickError
	return b
}
