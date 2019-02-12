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
