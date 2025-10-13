package concurrency

import (
	"time"
)

// Do performs the action as soon as the waitable is done.
// It blocks indefinitely until that happens.
func Do(w Waitable, action func()) {
	Wait(w)
	action()
}

// DoWithTimeout performs the action as soon as the waitable is done.
// It gives up and returns after timeout, and returns a bool indicating whether
// the action was performed or not.
func DoWithTimeout(w Waitable, action func(), timeout time.Duration) bool {
	if WaitWithTimeout(w, timeout) {
		action()
		return true
	}
	return false
}

// DoInWaitable performs an action asynchronously bound to a waitable.
// It blocks until either the action is performed, or the waitable is done,
// and returns the waitable's error in case the waitable is done first.
func DoInWaitable(w ErrorWaitable, action func()) error {
	var wg WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Add(-1)
		action()
	}()

	return WaitForErrorInContext(w, &wg)
}
