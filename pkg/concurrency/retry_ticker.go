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
	Stop()
}

type tickFunc func(ctx context.Context) (timeToNextTick time.Duration, err error)

// NewRetryTicker returns a new RetryTicker that calls the "tick function" `doFunc` repeatedly:
// - The RetryTicker calls `doFunc` immediately, and if that returns an error
// then the RetryTicker will wait the time returned by `backoff.Step` before calling `doFunc` again.
// - `doFunc` should return an error if ctx is cancelled. RetryTicker always calls `doFunc` with a context
// with a timeout of `timeout`.
// - On success `RetryTicker` will reset `backoff`, and wait the amount of time returned by `doFunc` before
// running it again.
func NewRetryTicker(doFunc tickFunc, timeout time.Duration, backoff wait.Backoff) RetryTicker {
	return newRetryTicker(doFunc, timeout, backoff, true)
}

func newRetryTicker(doFunc tickFunc, timeout time.Duration, backoff wait.Backoff, start bool) RetryTicker {
	ticker := &retryTickerImpl{
		scheduler:      time.AfterFunc,
		doFunc:         doFunc,
		timeout:        timeout,
		initialBackoff: backoff,
		backoff:        backoff,
	}
	if start {
		ticker.start()
	}
	return ticker
}

type retryTickerImpl struct {
	scheduler      func(d time.Duration, f func()) *time.Timer
	doFunc         tickFunc
	timeout        time.Duration
	initialBackoff wait.Backoff
	backoff        wait.Backoff
	timer          *time.Timer
	mutex          sync.RWMutex
}

// Start calls the tick function and schedules the next tick immediately.
func (t *retryTickerImpl) start() {
	t.scheduleTick(0)
}

// Stop cancels this RetryTicker. If Stop is called while the tick function is running then Stop does not
// wait for the tick function to complete before returning.
func (t *retryTickerImpl) Stop() {
	t.setTickTimer(nil)
}

func (t *retryTickerImpl) scheduleTick(timeToTick time.Duration) {
	t.setTickTimer(t.scheduler(timeToTick, func() {
		ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
		defer cancel()

		nextTimeToTick, tickErr := t.doFunc(ctx)
		if t.getTickTimer() == nil {
			// ticker was stopped while tick function was running.
			return
		}
		if tickErr != nil {
			t.scheduleTick(t.backoff.Step())
			return
		}
		// reset backoff strategy.
		t.backoff = t.initialBackoff
		t.scheduleTick(nextTimeToTick)
	}))
}

func (t *retryTickerImpl) setTickTimer(timer *time.Timer) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.timer != nil {
		t.timer.Stop()
	}
	t.timer = timer
}

func (t *retryTickerImpl) getTickTimer() *time.Timer {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.timer
}
