package queue

import (
	"context"
	"testing"
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
		for i := 0; i < numItems; i++ {
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
		for i := 0; i < numGoroutines; i++ {
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
}
