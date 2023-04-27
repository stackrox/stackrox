package concurrency

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	_ RetryTicker = (*retryTickerImpl)(nil)
	// ErrStartedTimer is returned when Start is called on a timer that was already started.
	ErrStartedTimer = errors.New("started timer")
	// ErrStoppedTimer is returned when Start is called on a timer that was stopped.
	ErrStoppedTimer = errors.New("stopped timer")
	// ErrNonRecoverable should be returned by the ticker function in order to stop this timer.
	ErrNonRecoverable = errors.New("non-recoverable error")
)

// RetryTicker repeatedly calls a function with a timeout and a retry backoff strategy.
// RetryTickers can only be started once.
// RetryTickers are not safe for simultaneous use by multiple goroutines.
type RetryTicker interface {
	Start() error
	Stop()
	Stopped() bool
}

type tickFunc func(ctx context.Context) (timeToNextTick time.Duration, err error)

// NewRetryTicker returns a new RetryTicker that calls the "tick function" `doFunc` repeatedly:
// - When started, the RetryTicker calls `doFunc` immediately, and if that returns an error
// then:
//   - If that error is ErrNonRecoverable (according to `errors.Is`) then the RetryTicker stops.
//   - Otherwise the RetryTicker will wait the time returned by `backoff.Step` before calling `doFunc` again.
//
// - `doFunc` should return an error if ctx is cancelled. RetryTicker always calls `doFunc` with a context
// with a timeout of `timeout`.
// - On success `RetryTicker` will reset `backoff`, and wait the amount of time returned by `doFunc` before
// running it again.
func NewRetryTicker(doFunc tickFunc, timeout time.Duration, backoff wait.Backoff) RetryTicker {
	return &retryTickerImpl{
		scheduler:      time.AfterFunc,
		doFunc:         doFunc,
		timeout:        timeout,
		initialBackoff: backoff,
		backoff:        backoff,
	}
}

type retryTickerImpl struct {
	scheduler      func(d time.Duration, f func()) *time.Timer
	doFunc         tickFunc
	timeout        time.Duration
	initialBackoff wait.Backoff
	backoff        wait.Backoff
	timer          *time.Timer
	timerMutex     sync.RWMutex
	stopFlag       Flag
}

// Start calls the tick function and schedules the next tick immediately.
// Start returns and error if the RetryTicker is started more than once:
// - ErrStartedTimer is returned if the timer was already started.
// - ErrStoppedTimer is returned if the timer was stopped.
func (t *retryTickerImpl) Start() error {
	if t.Stopped() {
		return ErrStoppedTimer
	}
	if t.getTickTimer() != nil {
		return ErrStartedTimer
	}
	t.backoff = t.initialBackoff // initialize backoff strategy
	t.scheduleTick(0)
	return nil
}

// Stop cancels this RetryTicker. If Stop is called while the tick function is running then Stop does not
// wait for the tick function to complete before returning.
func (t *retryTickerImpl) Stop() {
	t.stopFlag.Set(true)

	t.timerMutex.Lock()
	defer t.timerMutex.Unlock()
	t.setTickTimer(nil)
}

// Stopped returns true if this RetryTicker has been stopped, otherwise returns false.
func (t *retryTickerImpl) Stopped() bool {
	return t.stopFlag.Get()
}

func (t *retryTickerImpl) scheduleTick(timeToTick time.Duration) {
	t.timerMutex.Lock()
	defer t.timerMutex.Unlock()

	timer := t.scheduler(timeToTick, func() {
		ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
		defer cancel()

		nextTimeToTick, tickErr := t.doFunc(ctx)
		if t.Stopped() {
			// ticker was stopped while tick function was running.
			return
		}
		if errors.Is(tickErr, ErrNonRecoverable) {
			t.Stop()
			return
		}
		if tickErr != nil {
			t.scheduleTick(t.backoff.Step())
			return
		}
		t.backoff = t.initialBackoff // reset backoff strategy
		t.scheduleTick(nextTimeToTick)
	})
	// We take mutex at the start of this method (rather than inside setTickTimer)
	// to make sure the setTickTimer() call below completes before one of
	// the t.scheduleTick() calls in the goroutine started by t.scheduler() above
	// starts running.
	t.setTickTimer(timer)
}

func (t *retryTickerImpl) setTickTimer(timer *time.Timer) {
	if t.timer != nil {
		t.timer.Stop()
	}
	t.timer = timer
}

func (t *retryTickerImpl) getTickTimer() *time.Timer {
	t.timerMutex.RLock()
	defer t.timerMutex.RUnlock()
	return t.timer
}
