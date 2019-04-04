package concurrency

import "time"

// Wait waits indefinitely until the condition represented by the given Waitable is fulfilled.
func Wait(w Waitable) {
	<-w.Done()
}

// IsDone checks if the given waitable's condition is fulfilled.
func IsDone(w Waitable) bool {
	select {
	case <-w.Done():
		return true
	default:
		return false
	}
}

// WaitWithTimeout waits for the given Waitable with a specified timeout. It returns false if the timeout expired
// before the condition was fulfilled, true otherwise.
func WaitWithTimeout(w Waitable, timeout time.Duration) bool {
	if timeout <= 0 {
		return IsDone(w)
	}

	t := time.NewTimer(timeout)
	select {
	case <-w.Done():
		if !t.Stop() {
			<-t.C
		}
		return true
	case <-t.C:
		return false
	}
}

// WaitWithDeadline waits for the given Waitable until a specified deadline. It returns false if the deadline expired
// before the condition was fulfilled, true otherwise.
func WaitWithDeadline(w Waitable, deadline time.Time) bool {
	timeout := time.Until(deadline)
	return WaitWithTimeout(w, timeout)
}

// WaitInContext waits for the given Waitable until a `parentContext` is done. Note that despite its name,
// `parentContext` can be any waitable, not just a context.
// It returns false if the parentContext is done first, true otherwise.
func WaitInContext(w Waitable, parentContext Waitable) bool {
	select {
	case <-w.Done():
		return true
	case <-parentContext.Done():
		return false
	}
}
