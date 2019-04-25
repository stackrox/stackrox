package reprocessor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestThrottlesFastCalls(t *testing.T) {
	throttler := newThrottle(10 * time.Millisecond)

	// Run count should be two, one for the first, and one for the end of the window since more were called.
	var ran string
	throttler.run(func() {
		ran = ran + "1"
	})

	throttler.run(func() {
		ran = ran + "2"
	})

	throttler.run(func() {
		ran = ran + "3"
	})

	throttler.run(func() {
		ran = ran + "4"
	})

	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, "12", ran)
}

func TestThrottlesSlowCalls(t *testing.T) {
	throttler := newThrottle(10 * time.Millisecond)

	// Run count should be two, one for the first, and one for the end of the window since more were called.
	var ran string
	throttler.run(func() {
		time.Sleep(20 * time.Millisecond)
		ran = ran + "1"
	})

	throttler.run(func() {
		time.Sleep(20 * time.Millisecond)
		ran = ran + "2"
	})

	throttler.run(func() {
		time.Sleep(20 * time.Millisecond)
		ran = ran + "3"
	})

	throttler.run(func() {
		time.Sleep(20 * time.Millisecond)
		ran = ran + "4"
	})

	// Both should complete by 30 millis.
	// the first will run in 20, and the last will run in 20 after a 10 millisecond wait window)
	time.Sleep(40 * time.Millisecond)
	assert.Equal(t, "12", ran)
}
