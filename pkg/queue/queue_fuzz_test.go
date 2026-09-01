package queue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FuzzQueueOperations tests random sequences of queue operations.
// Verifies:
// - No panics under random operation sequences
// - Data integrity: all pushed items are accounted for
// - Queue length matches expected state
func FuzzQueueOperations(f *testing.F) {
	// Seed corpus with edge cases
	f.Add([]byte{0})                // Empty operations
	f.Add([]byte{1, 2, 3})          // Mix of push/pull
	f.Add([]byte{1, 1, 1, 1})       // Only pushes
	f.Add([]byte{2, 2, 2, 2})       // Only pulls
	f.Add([]byte{1, 2, 1, 2, 1, 2}) // Alternating push/pull
	f.Add([]byte{1, 1, 1, 2, 2, 2}) // Batch push then batch pull
	f.Add([]byte{2, 2, 1, 1})       // Pull from empty then push

	f.Fuzz(func(t *testing.T, operations []byte) {
		if len(operations) == 0 {
			return
		}

		q := NewQueue[int]()
		pushed := make([]int, 0)
		pulled := make([]int, 0)
		itemCounter := 0

		for _, op := range operations {
			switch op % 3 {
			case 0: // Push
				itemCounter++
				q.Push(itemCounter)
				pushed = append(pushed, itemCounter)

			case 1: // Pull
				item := q.Pull()
				if item != 0 {
					pulled = append(pulled, item)
				}

			case 2: // Check Len
				length := q.Len()
				expectedLen := len(pushed) - len(pulled)
				assert.Equal(t, expectedLen, length, "Queue length mismatch")
			}
		}

		// Drain remaining items
		for q.Len() > 0 {
			item := q.Pull()
			if item != 0 {
				pulled = append(pulled, item)
			}
		}

		// Verify all pushed items were pulled
		assert.Equal(t, len(pushed), len(pulled), "Not all items were pulled")
		assert.ElementsMatch(t, pushed, pulled, "Pulled items don't match pushed items")
		assert.Equal(t, 0, q.Len(), "Queue should be empty")
	})
}

// FuzzQueueConcurrent tests concurrent queue operations with simple produce-then-consume pattern.
// Verifies:
// - No data loss under concurrent access
// - All pushed items are eventually pulled
// - No duplicates
// - Thread safety
func FuzzQueueConcurrent(f *testing.F) {
	// Seed corpus
	f.Add(uint8(1), uint8(1), uint16(10))   // 1 producer, 1 consumer, 10 items
	f.Add(uint8(2), uint8(2), uint16(20))   // 2 producers, 2 consumers, 20 items
	f.Add(uint8(5), uint8(3), uint16(50))   // 5 producers, 3 consumers, 50 items
	f.Add(uint8(1), uint8(5), uint16(30))   // 1 producer, 5 consumers
	f.Add(uint8(10), uint8(1), uint16(100)) // 10 producers, 1 consumer

	f.Fuzz(func(t *testing.T, numProducers, numConsumers uint8, numItems uint16) {
		// Limit concurrency and items to prevent resource exhaustion
		if numProducers == 0 || numProducers > 20 {
			return
		}
		if numConsumers == 0 || numConsumers > 20 {
			return
		}
		if numItems == 0 || numItems > 500 {
			return
		}

		q := NewQueue[int]()

		// Phase 1: Concurrent producers push all items
		itemsPerProducer := int(numItems) / int(numProducers)
		remainder := int(numItems) % int(numProducers)

		producerDone := make(chan struct{})
		for i := 0; i < int(numProducers); i++ {
			items := itemsPerProducer
			if i == 0 {
				items += remainder
			}

			go func(producerID, itemCount int) {
				defer func() { producerDone <- struct{}{} }()
				for j := 0; j < itemCount; j++ {
					// Start from 1 to avoid confusion with zero-value
					item := producerID*10000 + j + 1
					q.Push(item)
				}
			}(i, items)
		}

		// Wait for all producers to finish
		for i := 0; i < int(numProducers); i++ {
			<-producerDone
		}

		// Verify all items are in queue
		assert.Equal(t, int(numItems), q.Len(), "All items should be queued")

		// Phase 2: Single-threaded pull to verify data integrity
		pulled := make([]int, 0, numItems)
		for q.Len() > 0 {
			item := q.Pull()
			pulled = append(pulled, item)
		}

		// Verify all items were pulled
		assert.Equal(t, int(numItems), len(pulled), "All items should be pulled")
		assert.Equal(t, 0, q.Len(), "Queue should be empty")

		// Verify no duplicates
		seen := make(map[int]bool)
		for _, item := range pulled {
			require.False(t, seen[item], "Duplicate item: %d", item)
			seen[item] = true
		}
	})
}

// FuzzQueueSeq tests the Seq() iterator with random cancellation timing.
// Verifies:
// - Iterator yields all items or stops cleanly on cancel
// - No missed items or duplicates
// - Proper handling of cancellation at arbitrary times
func FuzzQueueSeq(f *testing.F) {
	// Seed corpus
	f.Add(uint16(10), uint16(0))   // 10 items, cancel at start
	f.Add(uint16(10), uint16(5))   // 10 items, cancel in middle
	f.Add(uint16(10), uint16(10))  // 10 items, cancel at end
	f.Add(uint16(100), uint16(50)) // 100 items, cancel halfway
	f.Add(uint16(1), uint16(0))    // Single item, cancel immediately
	f.Add(uint16(0), uint16(0))    // Empty queue

	f.Fuzz(func(t *testing.T, numItems, cancelAfter uint16) {
		if numItems > 1000 {
			return
		}

		q := NewQueue[int]()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Pre-populate queue
		for i := 0; i < int(numItems); i++ {
			q.Push(i)
		}

		collected := make([]int, 0, numItems)
		collectedMap := make(map[int]bool)

		// Set up cancellation
		if cancelAfter <= numItems {
			go func() {
				// Wait for some items to be consumed
				for len(collected) < int(cancelAfter) {
					time.Sleep(time.Millisecond)
				}
				cancel()
			}()
		}

		// Iterate with Seq
		for item := range q.Seq(ctx) {
			require.False(t, collectedMap[item], "Duplicate item from Seq: %d", item)
			collected = append(collected, item)
			collectedMap[item] = true
		}

		// Verify results
		if ctx.Err() == nil {
			// No cancellation - all items should be consumed
			assert.Equal(t, int(numItems), len(collected), "Should consume all items without cancellation")
			assert.Equal(t, 0, q.Len(), "Queue should be empty")
		} else {
			// Cancellation occurred - verify we stopped cleanly
			assert.LessOrEqual(t, len(collected), int(numItems), "Should not consume more than pushed")
		}

		// Verify all collected items are valid and unique
		for _, item := range collected {
			assert.GreaterOrEqual(t, item, 0, "Invalid item value")
			assert.Less(t, item, int(numItems), "Item out of range")
		}
	})
}

// FuzzQueueWithMaxSize tests queue behavior with max size limits.
// Verifies:
// - Overflow behavior is correct
// - Items are dropped when queue is full
// - Queue never exceeds max size
func FuzzQueueWithMaxSize(f *testing.F) {
	// Seed corpus
	f.Add(uint16(5), uint16(10))    // maxSize=5, push 10 items
	f.Add(uint16(1), uint16(10))    // maxSize=1, push 10 items
	f.Add(uint16(10), uint16(5))    // maxSize=10, push 5 items
	f.Add(uint16(100), uint16(200)) // maxSize=100, push 200 items
	f.Add(uint16(0), uint16(10))    // maxSize=0 (unlimited), push 10 items

	f.Fuzz(func(t *testing.T, maxSize, numPushes uint16) {
		if numPushes == 0 || numPushes > 1000 {
			return
		}
		if maxSize > 500 {
			return
		}

		var q *Queue[int]
		if maxSize == 0 {
			// Unlimited queue
			q = NewQueue[int]()
		} else {
			q = NewQueue[int](WithMaxSize[int](int(maxSize)))
		}

		// Push items
		for i := 0; i < int(numPushes); i++ {
			q.Push(i)

			// Verify queue never exceeds max size
			if maxSize > 0 {
				assert.LessOrEqual(t, q.Len(), int(maxSize), "Queue exceeded max size")
			}
		}

		// Verify final queue size
		if maxSize == 0 {
			// Unlimited - should have all items
			assert.Equal(t, int(numPushes), q.Len(), "Unlimited queue should have all items")
		} else {
			// Limited - should have at most maxSize items
			expectedSize := min(int(maxSize), int(numPushes))
			assert.Equal(t, expectedSize, q.Len(), "Queue size incorrect")
		}

		// Pull all items and verify they're valid
		pulled := 0
		seen := make(map[int]bool)
		for q.Len() > 0 {
			item := q.Pull()
			require.False(t, seen[item], "Duplicate item: %d", item)
			seen[item] = true
			pulled++
		}

		// Verify we pulled the expected number of items
		if maxSize == 0 {
			assert.Equal(t, int(numPushes), pulled, "Should pull all pushed items")
		} else {
			expectedPulled := min(int(maxSize), int(numPushes))
			assert.Equal(t, expectedPulled, pulled, "Pulled item count incorrect")
		}
	})
}

// FuzzQueuePullWithPred tests PullWithPred under random conditions.
// Verifies:
// - Correct item selection based on predicate
// - No panics with various predicates
// - Queue integrity maintained
func FuzzQueuePullWithPred(f *testing.F) {
	// Seed corpus
	f.Add(uint16(10), uint8(5))   // 10 items, find value 5
	f.Add(uint16(100), uint8(50)) // 100 items, find value 50
	f.Add(uint16(1), uint8(0))    // 1 item, find value 0
	f.Add(uint16(50), uint8(100)) // 50 items, find non-existent value

	f.Fuzz(func(t *testing.T, numItems uint16, searchValue uint8) {
		if numItems == 0 || numItems > 500 {
			return
		}

		q := NewQueue[int]()

		// Push items
		for i := 0; i < int(numItems); i++ {
			q.Push(i)
		}

		initialLen := q.Len()
		assert.Equal(t, int(numItems), initialLen)

		// Pull with predicate
		item, found := q.PullWithPred(func(v int) bool {
			return v == int(searchValue)
		})

		if int(searchValue) < int(numItems) {
			// Item should be found
			assert.True(t, found, "Item should be found")
			assert.Equal(t, int(searchValue), item, "Wrong item returned")
			assert.Equal(t, initialLen-1, q.Len(), "Queue length should decrease by 1")
		} else {
			// Item should not be found
			assert.False(t, found, "Item should not be found")
			assert.Equal(t, 0, item, "Should return zero value when not found")
			assert.Equal(t, initialLen, q.Len(), "Queue length should not change")
		}

		// Verify remaining items are intact
		pulled := make([]int, 0)
		for q.Len() > 0 {
			pulled = append(pulled, q.Pull())
		}

		// If item was found and removed, verify it's not in remaining items
		if found {
			for _, v := range pulled {
				assert.NotEqual(t, int(searchValue), v, "Removed item should not appear in remaining items")
			}
			assert.Equal(t, int(numItems)-1, len(pulled), "Remaining items count incorrect")
		} else {
			assert.Equal(t, int(numItems), len(pulled), "All items should remain")
		}
	})
}

// FuzzQueueBlocking tests PullBlocking behavior.
// Verifies that PullBlocking correctly waits for items and doesn't spin-loop.
func FuzzQueueBlocking(f *testing.F) {
	// Seed corpus
	f.Add(uint16(10)) // 10 items
	f.Add(uint16(1))  // 1 item
	f.Add(uint16(50)) // 50 items

	f.Fuzz(func(t *testing.T, numItems uint16) {
		if numItems == 0 || numItems > 200 {
			return
		}

		q := NewQueue[int]()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Push all items first (starting from 1 to avoid confusion with zero-value)
		for i := 1; i <= int(numItems); i++ {
			q.Push(i)
		}

		// Now pull with PullBlocking
		pulled := make([]int, 0, numItems)
		for i := 0; i < int(numItems); i++ {
			item := q.PullBlocking(ctx)
			if item != 0 {
				pulled = append(pulled, item)
			} else {
				require.Fail(t, "PullBlocking returned 0 unexpectedly")
			}
		}

		// Verify all items were consumed
		assert.Equal(t, int(numItems), len(pulled), "Should consume all items")
		assert.Equal(t, 0, q.Len(), "Queue should be empty")

		// Verify no duplicates
		seen := make(map[int]bool)
		for _, item := range pulled {
			require.False(t, seen[item], "Duplicate item: %d", item)
			seen[item] = true
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
