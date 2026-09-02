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

// testEmptyPullAfterSignal is a shared test helper that verifies a consumer properly
// re-blocks when signaled but another consumer steals the item. This prevents spin-loops.
func testEmptyPullAfterSignal(t *testing.T, consumerName string, startConsumer func(*Queue[int], context.Context, chan int, chan struct{})) {
	synctest.Test(t, func(t *testing.T) {
		q := NewQueue[int]()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		itemReceived := make(chan int, 1)
		consumerStarted := make(chan struct{})

		// Start consumer on empty queue
		startConsumer(q, ctx, itemReceived, consumerStarted)

		<-consumerStarted
		synctest.Wait() // Consumer is blocked waiting

		// Push an item, but immediately pull it with another consumer
		q.Push(99)
		pulled := q.Pull()
		assert.Equal(t, 99, pulled)

		// Consumer should remain blocked (not spin-loop) even though it was signaled
		synctest.Wait()

		select {
		case <-itemReceived:
			t.Fatalf("%s should not have received the pulled item", consumerName)
		default:
			// Expected: Consumer is still blocked
		}

		// Now push a second item - consumer should receive this one
		q.Push(42)
		synctest.Wait()

		select {
		case item := <-itemReceived:
			assert.Equal(t, 42, item)
		default:
			t.Fatal("expected to receive item from queue")
		}
	})
}

// testCancellationWithEmptyQueue is a shared test helper that verifies a consumer
// returns immediately when the context is already cancelled.
func testCancellationWithEmptyQueue(t *testing.T, consume func(context.Context, *Queue[int]) int) {
	synctest.Test(t, func(t *testing.T) {
		q := NewQueue[int]()

		// Create already-cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Verify context is cancelled
		select {
		case <-ctx.Done():
			// Expected: context is already done
		default:
			t.Fatal("context should be cancelled")
		}

		// Consumer should return immediately with zero value
		item := consume(ctx, q)
		assert.Equal(t, 0, item, "should return zero value when cancelled")
	})
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

	t.Run("Seq Empty Pull After Signal", func(t *testing.T) {
		// This test verifies that Seq() properly re-blocks when a signal occurs
		// but another consumer removes the item before Seq can pull it.
		// Without proper blocking, Seq would spin-loop after the empty pull.
		testEmptyPullAfterSignal(t, "Seq", func(q *Queue[int], ctx context.Context, itemReceived chan int, started chan struct{}) {
			go func() {
				close(started)
				for item := range q.Seq(ctx) {
					itemReceived <- item
					return
				}
			}()
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

	t.Run("Seq Lost Wakeup With Competing Consumers", func(t *testing.T) {
		// This test reproduces the lost wakeup bug with competing consumers.
		// Scenario: Multiple PullBlocking consumers compete with a Seq consumer.
		// When an item is pushed:
		// 1. Signal fires
		// 2. PullBlocking consumer wakes and pulls item (signal resets if queue empty)
		// 3. Seq consumer wakes but queue is empty, should re-wait
		// 4. Bug: Seq doesn't re-wait, it returns to top of loop and checks empty again
		//
		// With many iterations, we should see Seq miss items that were stolen.
		synctest.Test(t, func(t *testing.T) {
			q := NewQueue[int]()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			seqReceived := make(chan int, 100)
			pullReceived := make(chan int, 100)
			pullDone := make(chan struct{})

			const numItems = 10

			// Start Seq() consumer
			go func() {
				for item := range q.Seq(ctx) {
					seqReceived <- item
				}
			}()

			// Start competing PullBlocking consumer
			go func() {
				defer close(pullDone)
				for {
					item := q.PullBlocking(ctx)
					select {
					case <-ctx.Done():
						return
					default:
						pullReceived <- item
					}
				}
			}()

			synctest.Wait() // Let both consumers start and block

			// Push items one at a time
			for i := 1; i <= numItems; i++ {
				q.Push(i)
				synctest.Wait() // Let consumers race for the item
			}

			cancel() // Stop consumers
			synctest.Wait()

			<-pullDone
			close(seqReceived)
			close(pullReceived)

			seqItems := []int{}
			for item := range seqReceived {
				seqItems = append(seqItems, item)
			}

			pullItems := []int{}
			for item := range pullReceived {
				pullItems = append(pullItems, item)
			}

			// Both consumers should collectively receive all items
			allItems := append(seqItems, pullItems...)
			assert.Equal(t, numItems, len(allItems), "Total items received should equal items pushed")

			// Verify all items were received exactly once
			seen := make(map[int]bool)
			for _, item := range allItems {
				if seen[item] {
					t.Fatalf("Item %d received more than once", item)
				}
				seen[item] = true
			}

			for i := 1; i <= numItems; i++ {
				if !seen[i] {
					t.Fatalf("Item %d was lost - neither consumer received it", i)
				}
			}
		})
	})

	t.Run("Seq Stale Not Empty Signal Does Not Spin", func(t *testing.T) {
		testStaleNotEmptySignalDoesNotSpin(t, func(q *Queue[int], ctx context.Context) int {
			for item := range q.Seq(ctx) {
				return item
			}
			return 0
		})
	})

	t.Run("Seq Push Between Empty Pull And Reset Does Not Lose Item", func(t *testing.T) {
		testPushBetweenEmptyPullAndResetDoesNotLoseItem(t, func(q *Queue[int], ctx context.Context) int {
			for item := range q.Seq(ctx) {
				return item
			}
			return 0
		})
	})
}

func TestQueuePullBlocking(t *testing.T) {
	t.Run("PullBlocking Empty Pull After Signal", func(t *testing.T) {
		// This test verifies that PullBlocking() properly re-blocks when a signal
		// occurs but another consumer removes the item before PullBlocking can pull it.
		// Without proper blocking, PullBlocking would spin-loop after the empty pull.
		testEmptyPullAfterSignal(t, "PullBlocking", func(q *Queue[int], ctx context.Context, itemReceived chan int, started chan struct{}) {
			go func() {
				close(started)
				item := q.PullBlocking(ctx)
				itemReceived <- item
			}()
		})
	})

	t.Run("PullBlocking Cancellation", func(t *testing.T) {
		// This test verifies that PullBlocking() checks waitable.Done()
		// before blocking, not just after. If the waitable is already cancelled,
		// PullBlocking should return immediately.
		testCancellationWithEmptyQueue(t, func(ctx context.Context, q *Queue[int]) int {
			return q.PullBlocking(ctx)
		})
	})

	t.Run("PullBlocking Stale Not Empty Signal Does Not Spin", func(t *testing.T) {
		testStaleNotEmptySignalDoesNotSpin(t, func(q *Queue[int], ctx context.Context) int {
			return q.PullBlocking(ctx)
		})
	})

	t.Run("PullBlocking Push Between Empty Pull And Reset Does Not Lose Item", func(t *testing.T) {
		testPushBetweenEmptyPullAndResetDoesNotLoseItem(t, func(q *Queue[int], ctx context.Context) int {
			return q.PullBlocking(ctx)
		})
	})
}

// testStaleNotEmptySignalDoesNotSpin covers an empty queue whose notEmptySignal is
// already triggered. Waiters must Reset that latch and block; otherwise synctest.Wait hangs.
func testStaleNotEmptySignalDoesNotSpin(t *testing.T, consume func(*Queue[int], context.Context) int) {
	synctest.Test(t, func(t *testing.T) {
		q := NewQueue[int]()
		q.notEmptySignal.Signal()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		got := make(chan int, 1)
		started := make(chan struct{})
		go func() {
			close(started)
			got <- consume(q, ctx)
		}()
		<-started
		synctest.Wait()
		t.Logf("consumer blocked")
		q.Push(7)
		synctest.Wait()
		select {
		case item := <-got:
			assert.Equal(t, 7, item)
		default:
			t.Fatal("consumer should have received the item after blocking")
		}
	})
}

// testPushBetweenEmptyPullAndResetDoesNotLoseItem covers the pullWait window
// between an empty pull() and Reset(). synctest cannot preempt there, so
// afterEmptyPull injects the Push.
func testPushBetweenEmptyPullAndResetDoesNotLoseItem(t *testing.T, consume func(*Queue[int], context.Context) int) {
	synctest.Test(t, func(t *testing.T) {
		q := NewQueue[int]()
		q.afterEmptyPull = func() {
			q.Push(7)
			q.afterEmptyPull = nil
		}

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		got := make(chan int, 1)
		started := make(chan struct{})
		go func() {
			close(started)
			got <- consume(q, ctx)
		}()
		<-started
		synctest.Wait()

		select {
		case item := <-got:
			assert.Equal(t, 7, item)
		default:
			t.Fatal("Push between empty pull and Reset was lost; item is queued but the waiter is blocked")
		}
	})
}
