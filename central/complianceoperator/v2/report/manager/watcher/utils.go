package watcher

import "time"

// Timer interface to wrap a timer, so it can be injected in tests
type Timer interface {
	Stop() bool
	C() <-chan time.Time
	Reset()
}

type timerWrapper struct {
	timer    *time.Timer
	duration time.Duration
}

// NewTimer creates a new timerWrapper
func NewTimer(duration time.Duration) *timerWrapper {
	return &timerWrapper{
		timer:    time.NewTimer(duration),
		duration: duration,
	}
}

// Stop the timer
func (t *timerWrapper) Stop() bool {
	return t.timer.Stop()
}

// C returns the timer channel
func (t *timerWrapper) C() <-chan time.Time {
	return t.timer.C
}

// Reset the timer
func (t *timerWrapper) Reset() {
	t.timer.Reset(t.duration)
}
