package concurrency

import (
	"time"
)

// Timeout returns a waitable that is done after the given timeout.
// Note: Every call to `Timeout` with a positive duration spawns a new goroutine that lives for the given duration,
// hence this function should not be used in potentially hot code sites. If possible, prefer `TimeoutOr`.
func Timeout(duration time.Duration) WaitableChan {
	if duration <= 0 {
		return closedCh
	}

	ch := make(chan struct{})
	go func() {
		time.Sleep(duration)
		close(ch)
	}()

	return ch
}

// TimeoutOr returns a waitable that gets triggered once the given timeout expires. If cancelCond is triggered
// beforehand, the returned waitable will never triggered.
// If possible, use this over `Timeout`, as the latter will cause goroutines to stick around for the entire timeout
// duration.
// The motivation for not triggering the returned waitable when cancelCond is triggered is to allow statements such as
//
//	select {
//	case <-sig.Done():
//		... happy path ...
//	case <-TimeoutOr(timeoutDuration, &sig):
//		... oh no, a timeout ...
//	}
//
// Due to the non-deterministic nature of select, a trigger might otherwise show up as a timeout, and only checking
// `sig.Done()` would allow a user to distinguish the two.
func TimeoutOr(duration time.Duration, cancelCond Waitable) WaitableChan {
	if duration <= 0 {
		return closedCh
	}

	t := time.NewTimer(duration)
	ch := make(chan struct{})

	go func() {
		select {
		case <-t.C:
			close(ch)
		case <-cancelCond.Done():
			if !t.Stop() {
				<-t.C
			}
		}
	}()

	return ch
}

// Deadline returns a waitable that is done after the given deadline.
// Note: Every call to `Deadline` with a future time point spawns a new goroutine that lives until the given deadline,
// hence this function should not be used in potentially hot code sites. If possible, prefer `DeadlineOr`.
func Deadline(deadline time.Time) WaitableChan {
	return Timeout(time.Until(deadline))
}

// DeadlineOr returns a waitable that is triggered when the given deadline expires, unless cancelCond is triggered
// beforehand.
func DeadlineOr(deadline time.Time, cancelCond Waitable) WaitableChan {
	return TimeoutOr(time.Until(deadline), cancelCond)
}
