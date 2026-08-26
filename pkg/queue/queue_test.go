package queue

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

func TestQueue(t *testing.T) {
	q := NewQueue[*string]()

	// 1. Adding a new item to the queue.
	item := "first-item"
	q.Push(&item)

	// 2. Using pull should retrieve the previously added item.
	queueItem := q.Pull()
	assert.Equal(t, item, *queueItem)

	// 3. Add an item after 500ms of waiting. Meanwhile, call pull blocking. It should wait until an item is added
	// and afterward return it.
	time.AfterFunc(500*time.Millisecond, func() {
		item := "second-item"
		q.Push(&item)
	})

	assert.Eventually(t, func() bool {
		queueItem := q.PullBlocking(context.Background())
		return "second-item" == *queueItem
	}, 1*time.Second, 100*time.Millisecond)

	// 4. Another pull should now return an empty value.
	queueItem = q.Pull()
	assert.Nil(t, queueItem)

	// 5. Empty element should be available to pull
	q.Push(nil)
	assert.Nil(t, q.PullBlocking(context.Background()))
}

func TestPullWithPred(t *testing.T) {
	q := NewQueue[int]()
	q.Push(1)
	q.Push(2)
	q.Push(3)
	q.Push(4)

	item, ok := q.PullWithPred(func(v int) bool { return v%2 == 0 })
	assert.True(t, ok)
	assert.Equal(t, 2, item)
	assert.Equal(t, 3, q.Len())

	item, ok = q.PullWithPred(func(v int) bool { return v > 3 })
	assert.True(t, ok)
	assert.Equal(t, 4, item)
	assert.Equal(t, 2, q.Len())

	_, ok = q.PullWithPred(func(v int) bool { return v > 100 })
	assert.False(t, ok)
	assert.Equal(t, 2, q.Len())

	// Remaining items are 1, 3
	assert.Equal(t, 1, q.Pull())
	assert.Equal(t, 3, q.Pull())
	assert.Equal(t, 0, q.Len())
}

func TestPullWithPredEmptyQueue(t *testing.T) {
	q := NewQueue[int]()
	_, ok := q.PullWithPred(func(v int) bool { return true })
	assert.False(t, ok)
}

func TestQueueSeq(t *testing.T) {
	t.Run("Basic Iteration", func(t *testing.T) {
		q := NewQueue[int]()

		q.Push(1)
		q.Push(2)
		q.Push(3)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		items := make([]int, 0, 3)
		for item := range q.Seq(ctx) {
			items = append(items, item)
			if len(items) == cap(items) {
				cancel()
			}
		}

		assert.Equal(t, []int{1, 2, 3}, items)
		assert.Equal(t, 0, q.Len())
	})

	t.Run("Seq Async Items", func(t *testing.T) {
		q := NewQueue[int]()

		expectedItems := []int{4, 5, 6}
		items := make([]int, 0, len(expectedItems))
		itemsAdded := make(chan struct{})
		itemsRead := make(chan struct{})
		ctx := context.Background()

		go func() {
			for item := range q.Seq(ctx) {
				items = append(items, item)
				if len(items) == len(expectedItems) {
					break
				}
			}
			close(itemsRead)
		}()

		go func() {
			for _, item := range expectedItems {
				q.Push(item)
			}
			close(itemsAdded)
		}()

		<-itemsAdded
		<-itemsRead

		assert.Equal(t, expectedItems, items)
	})

	t.Run("Seq Cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		q := NewQueue[int]()

		items := make([]int, 0)
		for item := range q.Seq(ctx) {
			items = append(items, item)
		}
		assert.Empty(t, items)
	})

	t.Run("Seq Concurrent Iteration", func(t *testing.T) {
		q := NewQueue[int]()

		// Add items to the queue
		numItems := 30
		results := make(chan int, numItems)
		expectedItems := make([]int, 0, numItems)
		for i := range numItems {
			q.Push(i)
			expectedItems = append(expectedItems, i)
		}

		// Create multiple goroutines that will iterate over the queue
		numGoroutines := 3
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			for q.Len() > 0 {
			}
			cancel()
		}()

		wg := sync.WaitGroup{}
		wg.Add(numGoroutines)
		for i := range numGoroutines {
			go func(goroutineID int) {
				defer wg.Done()

				for item := range q.Seq(ctx) {
					results <- item
				}
			}(i)
		}
		wg.Wait()
		close(results)

		// Collect all items from all goroutines
		items := make([]int, 0, numItems)
		for item := range results {
			items = append(items, item)
		}

		assert.ElementsMatch(t, expectedItems, items)

	})

	t.Run("Seq Empty Queue No Spin Loop", func(t *testing.T) {
		// This test verifies the fix for the spin-loop bug where Seq() would
		// continuously loop without blocking when the queue was empty.
		// Under synctest's deterministic execution, a spin-loop causes timeouts.
		synctest.Test(t, func(t *testing.T) {
			q := NewQueue[int]()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			itemReceived := make(chan int, 1)
			iterationStarted := make(chan struct{})

			// Start Seq() iteration on empty queue
			go func() {
				close(iterationStarted)
				for item := range q.Seq(ctx) {
					itemReceived <- item
					return
				}
			}()

			<-iterationStarted

			// Wait to ensure the goroutine is properly blocked (not spinning)
			// In the buggy version, this would cause synctest to detect a spin-loop
			synctest.Wait()

			// Now add an item - it should be received
			q.Push(42)
			synctest.Wait()

			select {
			case item := <-itemReceived:
				assert.Equal(t, 42, item)
			default:
				t.Fatal("expected to receive item from queue")
			}
		})
	})

	t.Run("Seq Cancellation With Queued Items", func(t *testing.T) {
		// This test verifies that Seq() checks waitable.Done() before every pull,
		// not just when the queue is empty. If items are already queued and the
		// waitable is cancelled, the iterator should exit immediately without
		// consuming any items.
		synctest.Test(t, func(t *testing.T) {
			q := NewQueue[int]()

			// Pre-populate queue with items
			q.Push(1)
			q.Push(2)
			q.Push(3)

			// Create already-cancelled context
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			// Verify context is cancelled before iteration starts
			select {
			case <-ctx.Done():
				// Expected: context is already done
			default:
				t.Fatal("context should be cancelled before iteration")
			}

			// Attempt to iterate - should exit immediately without consuming items
			itemsConsumed := 0
			for range q.Seq(ctx) {
				itemsConsumed++
			}

			// Assertions
			assert.Equal(t, 0, itemsConsumed, "should not consume any items when cancelled")
			assert.Equal(t, 3, q.Len(), "all items should remain in queue")
		})
	})
}
