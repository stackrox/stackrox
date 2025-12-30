package testutils

import (
	"testing"
	"time"
)

// Eventually works similarly to assert.Eventually, but it does not call
// t.Fail() when the deadline expires. This is useful for when retrying
// calls to Eventually.
func Eventually(t *testing.T, condition func() bool, waitFor, tick time.Duration) bool {
	ch := make(chan bool, 1)
	checkCond := func() { ch <- condition() }

	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	var tickC <-chan time.Time

	// Check the condition once first on the initial call.
	go checkCond()

	for {
		select {
		case <-timer.C:
			return false
		case <-tickC:
			tickC = nil
			go checkCond()
		case v := <-ch:
			if v {
				return true
			}
			tickC = ticker.C
		}
	}
}
