package timeutil

import "time"

// TimerC obtains the time channel from a timer. This function is `nil`-safe: in case of a `nil` timer, a `nil` channel
// is returned (which is legal to read from, but reading will block forever).
func TimerC(t *time.Timer) <-chan time.Time {
	if t == nil {
		return nil
	}
	return t.C
}

// StopTimer stops the given timer, taking care of any events that are still emitted due to a race condition.
// Note: This function must not be used if another active goroutine could read from the timer channel; otherwise,
// this function might deadlock.
func StopTimer(t *time.Timer) {
	if t != nil {
		if !t.Stop() {
			<-t.C
		}
	}
}
