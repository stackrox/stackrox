package concurrency

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	_ RetryTicker = (*retryTickerImpl)(nil)
)

// RetryTicker repeatedly calls a function with a timeout and a retry backoff strategy.
type RetryTicker interface {
	Start()
	Stop()
}

type retryTickerImpl struct {
	fn               tickFunc
	tickTimeout      time.Duration
	backoffPrototype wait.Backoff
	backoff          wait.Backoff
	tickTimer        *time.Timer
	tickTimerM       sync.Mutex
}

type tickFunc func(ctx context.Context) (timeToNextTick time.Duration, err error)
type onTickSuccessFunc func(nextTimeToTick time.Duration)
type onTickErrorFunc func(tickErr error)

// Start calls t.f and schedules the next tick immediately.
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

		nextTimeToTick, tickErr := t.fn(ctx)
		if tickErr != nil {
			t.scheduleTick(t.backoff.Step())
			return
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
func NewRetryTicker(fn tickFunc, tickTimeout time.Duration, backoff wait.Backoff) RetryTicker {
	return &retryTickerImpl{
		fn:               fn,
		tickTimeout:      tickTimeout,
		backoffPrototype: backoff,
		backoff:          backoff,
	}
}
