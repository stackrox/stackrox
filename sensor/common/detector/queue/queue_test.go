package queue

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
)

func createAndStartQueue(t *testing.T, stopper concurrency.Stopper, size int) *Queue[*string] {
	t.Helper()
	q := NewQueue[*string](stopper, "queue", size, nil, nil)
	q.Start()
	return q
}

func push(t *testing.T, q *Queue[*string]) {
	t.Helper()
	old := q.queue.Len()
	item := "item"
	q.Push(&item)
	assert.Equal(t, old+1, q.queue.Len())
}

func pause(_ *testing.T, q *Queue[*string]) {
	q.Pause()
}

func resume(_ *testing.T, q *Queue[*string]) {
	q.Resume()
}

// noPull verifies that Pull() blocks when the queue is paused.
// With synctest's fake clock, time advances only when all goroutines are durably blocked.
func noPull(t *testing.T, q *Queue[*string], stopper concurrency.Stopper) {
	t.Helper()
	ch := make(chan *string)
	go func() {
		defer close(ch)
		select {
		case item := <-q.Pull():
			// Pull returned - either got an item or queue was stopped (nil).
			// Use nested select to avoid blocking if nobody reads from ch.
			if item != nil {
				select {
				case ch <- item:
				case <-stopper.Flow().StopRequested():
				}
			}
		case <-stopper.Flow().StopRequested():
		}
	}()
	// With fake clock, this advances time instantly when goroutines are blocked
	select {
	case <-time.After(500 * time.Millisecond):
		return // Expected: timeout means Pull() is blocked
	case item := <-ch:
		t.Fatalf("should not pull from the queue, but %s was pulled", *item)
	}
}

// pull verifies that Pull() succeeds within a timeout.
func pull(t *testing.T, q *Queue[*string], stopper concurrency.Stopper) {
	t.Helper()
	ch := make(chan *string)
	go func() {
		defer close(ch)
		select {
		case item := <-q.Pull():
			// Use nested select to avoid blocking if nobody reads from ch.
			select {
			case ch <- item:
			case <-stopper.Flow().StopRequested():
			}
		case <-stopper.Flow().StopRequested():
		}
	}()
	select {
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting to pull from the queue")
	case item := <-ch:
		assert.Equal(t, "item", *item)
	}
}

// stopPull verifies that Pull() returns nil when the queue is stopped.
func stopPull(t *testing.T, q *Queue[*string]) {
	t.Helper()
	// Schedule the stop after a delay - with fake clock this fires when goroutines block
	time.AfterFunc(500*time.Millisecond, func() {
		q.stopper.Client().Stop()
	})
	// Wait for the goroutine to be blocked, then time advances and AfterFunc fires
	item := <-q.Pull()
	assert.Nil(t, item, "Pull() should return nil after stop")
}

// pushPullBlocking verifies that Pull() blocks until an item is pushed.
func pushPullBlocking(t *testing.T, q *Queue[*string]) {
	t.Helper()
	// Schedule a push after a delay - with fake clock this fires when goroutines block
	time.AfterFunc(500*time.Millisecond, func() {
		item := "item"
		q.Push(&item)
	})
	// Pull will block, time advances, AfterFunc fires, then Pull returns
	item := <-q.Pull()
	assert.NotNil(t, item)
	assert.Equal(t, "item", *item)
}

// TestPauseAndResume tests various queue pause/resume scenarios using synctest
// for deterministic concurrent testing with a fake clock.
func TestPauseAndResume(t *testing.T) {
	cases := map[string]func(t *testing.T){
		"Pause": func(t *testing.T) {
			testStopper := concurrency.NewStopper()
			queueStopper := concurrency.NewStopper()
			q := createAndStartQueue(t, queueStopper, 0)

			push(t, q)
			pause(t, q)
			noPull(t, q, testStopper)

			testStopper.Client().Stop()
			synctest.Wait()
			queueStopper.Client().Stop()
		},
		"Pause, resume": func(t *testing.T) {
			testStopper := concurrency.NewStopper()
			queueStopper := concurrency.NewStopper()
			q := createAndStartQueue(t, queueStopper, 0)

			push(t, q)
			pause(t, q)
			push(t, q)
			resume(t, q)
			pull(t, q, testStopper)
			pull(t, q, testStopper)

			testStopper.Client().Stop()
			synctest.Wait()
			queueStopper.Client().Stop()
		},
		"Pause, stop": func(t *testing.T) {
			testStopper := concurrency.NewStopper()
			queueStopper := concurrency.NewStopper()
			q := createAndStartQueue(t, queueStopper, 0)

			push(t, q)
			pause(t, q)
			noPull(t, q, testStopper)
			stopPull(t, q)

			testStopper.Client().Stop()
			synctest.Wait()
			queueStopper.Client().Stop()
		},
		"2 push, pull, pause, pull": func(t *testing.T) {
			testStopper := concurrency.NewStopper()
			queueStopper := concurrency.NewStopper()
			q := createAndStartQueue(t, queueStopper, 0)

			push(t, q)
			push(t, q)
			resume(t, q)
			pull(t, q, testStopper)
			synctest.Wait() // Replace time.Sleep - wait for goroutines to settle
			pause(t, q)
			pull(t, q, testStopper)
			stopPull(t, q)

			testStopper.Client().Stop()
			synctest.Wait()
			queueStopper.Client().Stop()
		},
		"2 Push, pull, pause, push, no pull": func(t *testing.T) {
			testStopper := concurrency.NewStopper()
			queueStopper := concurrency.NewStopper()
			q := createAndStartQueue(t, queueStopper, 0)

			push(t, q)
			push(t, q)
			resume(t, q)
			pull(t, q, testStopper)
			synctest.Wait() // Replace time.Sleep - wait for goroutines to settle
			pause(t, q)
			pull(t, q, testStopper)
			push(t, q)
			noPull(t, q, testStopper)
			stopPull(t, q)

			testStopper.Client().Stop()
			synctest.Wait()
			queueStopper.Client().Stop()
		},
		"Block until push": func(t *testing.T) {
			queueStopper := concurrency.NewStopper()
			q := createAndStartQueue(t, queueStopper, 0)

			resume(t, q)
			pushPullBlocking(t, q)

			queueStopper.Client().Stop()
		},
	}

	for name, testFn := range cases {
		t.Run(name, func(t *testing.T) {
			// Each subtest runs in its own synctest bubble
			synctest.Test(t, testFn)
		})
	}
}
