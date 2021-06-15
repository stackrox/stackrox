package throttle

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

// maybeRunAndTrack submits a function for execution through a throttler. If the submission succeeds,
// the run is tracked through the waitgroup.
func maybeRunAndTrack(throttler DropThrottle, wg *sync.WaitGroup, f func()) {
	// We need to increment the waitgroup before submitting for execution. Would we do it afterwards,
	// the run might already have completed and the waitgroup counter might run below 0.
	wg.Add(1)
	willRun := throttler.Run(func() {
		defer wg.Done()
		f()
	})
	if !willRun {
		wg.Done()
	}
}

func TestThrottlesFastCalls(t *testing.T) {
	t.Parallel()

	throttler := NewDropThrottle(500 * time.Millisecond)

	// Run count should be two, one for the first, and one for the end of the window since more were called.
	var ran string
	var ranMutex sync.Mutex
	var wg sync.WaitGroup
	maybeRunAndTrack(throttler, &wg, func() {
		concurrency.WithLock(&ranMutex, func() {
			ran = ran + "1"
		})
	})

	maybeRunAndTrack(throttler, &wg, func() {
		concurrency.WithLock(&ranMutex, func() {
			ran = ran + "2"
		})
	})

	maybeRunAndTrack(throttler, &wg, func() {
		concurrency.WithLock(&ranMutex, func() {
			ran = ran + "3"
		})
	})

	maybeRunAndTrack(throttler, &wg, func() {
		concurrency.WithLock(&ranMutex, func() {
			ran = ran + "4"
		})
	})

	wg.Wait()

	ranMutex.Lock()
	defer ranMutex.Unlock()
	assert.Equal(t, "12", ran)
}

func TestThrottlesSlowCalls(t *testing.T) {
	t.Parallel()

	throttler := NewDropThrottle(500 * time.Millisecond)

	// Run count should be two, one for the first, and one for the end of the window since more were called.
	var ran string
	var ranMutex sync.Mutex
	var wg sync.WaitGroup
	maybeRunAndTrack(throttler, &wg, func() {
		time.Sleep(200 * time.Millisecond)
		concurrency.WithLock(&ranMutex, func() {
			ran = ran + "1"
		})
	})

	maybeRunAndTrack(throttler, &wg, func() {
		time.Sleep(200 * time.Millisecond)
		concurrency.WithLock(&ranMutex, func() {
			ran = ran + "2"
		})
	})

	maybeRunAndTrack(throttler, &wg, func() {
		time.Sleep(200 * time.Millisecond)
		concurrency.WithLock(&ranMutex, func() {
			ran = ran + "3"
		})
	})

	maybeRunAndTrack(throttler, &wg, func() {
		time.Sleep(200 * time.Millisecond)
		concurrency.WithLock(&ranMutex, func() {
			ran = ran + "4"
		})
	})

	wg.Wait()

	ranMutex.Lock()
	defer ranMutex.Unlock()
	assert.Equal(t, "12", ran)
}
