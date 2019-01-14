package concurrency

import "time"

// Timeout returns a waitable that is done after the given timeout.
// Note: Every call to `Timeout` with a positive duration spawns a new goroutine that lives for the given duration,
// hence this function should not be used in potentially hot code sites.
func Timeout(duration time.Duration) Waitable {
	if duration <= 0 {
		return WaitableChan(closedCh)
	}

	ch := make(chan struct{})
	go func() {
		time.Sleep(duration)
		close(ch)
	}()

	return WaitableChan(ch)
}

// Deadline returns a waitable that is done after the given deadline.
// Note: Every call to `Deadline` with a future time point spawns a new goroutine that lives until the given deadline,
// hence this function should not be used in potentially hot code sites.
func Deadline(deadline time.Time) Waitable {
	return Timeout(deadline.Sub(time.Now()))
}
