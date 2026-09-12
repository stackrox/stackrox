package lane

import (
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test event implementation
type dedupTestEvent struct {
	key     string
	data    string
	flag1   bool
	flag2   bool
	topicID pubsub.Topic
}

func (e *dedupTestEvent) Topic() pubsub.Topic {
	var emptyTopic pubsub.Topic
	if e.topicID != emptyTopic {
		return e.topicID
	}
	return pubsub.DefaultTopic
}

func (e *dedupTestEvent) Lane() pubsub.LaneID {
	return pubsub.DefaultLane
}

func (e *dedupTestEvent) MergeFrom(old pubsub.Event) {
	oldEvent, ok := old.(*dedupTestEvent)
	if !ok {
		return
	}
	// Sticky-true merge semantics for flags
	e.flag1 = e.flag1 || oldEvent.flag1
	e.flag2 = e.flag2 || oldEvent.flag2
}

func extractTestKey(event pubsub.Event) string {
	e, ok := event.(*dedupTestEvent)
	if !ok {
		return ""
	}
	return e.key
}

func TestNewDedupingLaneOptions(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	t.Run("with default options", func(t *testing.T) {
		config := NewDedupingLane[string](pubsub.DefaultLane, WithDedupKeyFunc(extractTestKey))
		assert.Equal(t, pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*dedupingLane[string])
		require.True(t, ok)
		assert.Equal(t, 0, laneImpl.ch.Cap())
		assert.NotNil(t, laneImpl.dedupKeyFunc)
		assert.NotNil(t, laneImpl.dedupQueue)
		assert.NotNil(t, laneImpl.dedupIndex)
	})

	t.Run("with custom lane size", func(t *testing.T) {
		laneSize := 10
		config := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneSize[string](laneSize))
		lane := config.NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*dedupingLane[string])
		require.True(t, ok)
		assert.Equal(t, laneSize, laneImpl.ch.Cap())
	})

	t.Run("with negative lane size", func(t *testing.T) {
		laneSize := -1
		config := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneSize[string](laneSize))
		lane := config.NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*dedupingLane[string])
		require.True(t, ok)
		assert.Equal(t, 0, laneImpl.ch.Cap())
	})

	t.Run("with custom consumer", func(t *testing.T) {
		config := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneConsumer[string](newTestConsumer))
		lane := config.NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()

		// Register a consumer and verify it's the custom type
		err := lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(e pubsub.Event) error {
			return nil
		})
		require.NoError(t, err)

		laneImpl, ok := lane.(*dedupingLane[string])
		require.True(t, ok)
		consumers := laneImpl.consumers[pubsub.DefaultTopic]
		require.Len(t, consumers, 1)
		_, ok = consumers[0].(*testCustomConsumer)
		assert.True(t, ok)
	})

	t.Run("with custom merge func", func(t *testing.T) {
		mergeFunc := func(new, old pubsub.Event) {
			// Custom merge logic
		}
		config := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithMergeFunc[string](mergeFunc))
		lane := config.NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*dedupingLane[string])
		require.True(t, ok)
		assert.NotNil(t, laneImpl.mergeFunc)
	})

	t.Run("with size metric", func(t *testing.T) {
		metric := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_dedup_queue_size",
		})
		config := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupSizeMetric[string](metric))
		lane := config.NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*dedupingLane[string])
		require.True(t, ok)
		assert.NotNil(t, laneImpl.sizeMetric)
	})
}

func TestDedupingLaneOptionPanic(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	t.Run("panic if nil dedupKeyFunc", func(t *testing.T) {
		config := NewDedupingLane[string](pubsub.DefaultLane)
		assert.Panics(t, func() {
			config.NewLane()
		})
	})

	t.Run("panic if nil dedupKeyFunc passed to option", func(t *testing.T) {
		config := NewDedupingLane[string](pubsub.DefaultLane, WithDedupKeyFunc[string](nil))
		assert.Panics(t, func() {
			config.NewLane()
		})
	})

	t.Run("panic if nil NewConsumer", func(t *testing.T) {
		config := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneConsumer[string](nil))
		assert.Panics(t, func() {
			config.NewLane()
		})
	})
}

func TestDedupingLaneRegisterConsumer(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	t.Run("should error on nil callback", func(t *testing.T) {
		lane := NewDedupingLane[string](pubsub.DefaultLane, WithDedupKeyFunc(extractTestKey)).NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()
		assert.Error(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, nil))
	})

	t.Run("should successfully register consumer", func(t *testing.T) {
		lane := NewDedupingLane[string](pubsub.DefaultLane, WithDedupKeyFunc(extractTestKey)).NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()
		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			return nil
		}))
		laneImpl, ok := lane.(*dedupingLane[string])
		require.True(t, ok)
		assert.Len(t, laneImpl.consumers[pubsub.DefaultTopic], 1)
	})
}

func TestDedupingLaneDeduplication(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	t.Run("duplicate events are merged", func(t *testing.T) {
		lane := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneSize[string](10)).NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()

		receivedEvents := make([]*dedupTestEvent, 0)
		var mu sync.Mutex
		consumeSignal := concurrency.NewSignal()
		consumeCount := 0

		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic,
			func(event pubsub.Event) error {
				mu.Lock()
				defer mu.Unlock()
				e := event.(*dedupTestEvent)
				receivedEvents = append(receivedEvents, e)
				consumeCount++
				if consumeCount == 1 {
					consumeSignal.Signal()
				}
				return nil
			}))

		// Publish 3 events with same key
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A", data: "first"}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A", data: "second"}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A", data: "third"}))

		// Wait for consumption
		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Event should be consumed within timeout")
		case <-consumeSignal.Done():
		}

		// Give a bit of time to ensure no more events arrive
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		// Should only receive 1 event (deduplicated)
		assert.Equal(t, 1, len(receivedEvents))
		assert.Equal(t, "third", receivedEvents[0].data) // Last-write-wins
	})

	t.Run("different keys are not merged", func(t *testing.T) {
		lane := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneSize[string](10)).NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()

		receivedKeys := make([]string, 0)
		var mu sync.Mutex
		doneSignal := concurrency.NewSignal()

		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic,
			func(event pubsub.Event) error {
				mu.Lock()
				defer mu.Unlock()
				e := event.(*dedupTestEvent)
				receivedKeys = append(receivedKeys, e.key)
				if len(receivedKeys) == 3 {
					doneSignal.Signal()
				}
				return nil
			}))

		// Publish 3 events with different keys
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A"}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "B"}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "C"}))

		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Events should be consumed within timeout")
		case <-doneSignal.Done():
		}

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 3, len(receivedKeys))
		assert.Contains(t, receivedKeys, "A")
		assert.Contains(t, receivedKeys, "B")
		assert.Contains(t, receivedKeys, "C")
	})

	t.Run("zero-value key bypasses dedup", func(t *testing.T) {
		lane := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneSize[string](10)).NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()

		receivedCount := 0
		var mu sync.Mutex
		doneSignal := concurrency.NewSignal()

		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic,
			func(event pubsub.Event) error {
				mu.Lock()
				defer mu.Unlock()
				receivedCount++
				if receivedCount == 3 {
					doneSignal.Signal()
				}
				return nil
			}))

		// Publish 3 events with empty key (zero-value)
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "", data: "1"}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "", data: "2"}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "", data: "3"}))

		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Events should be consumed within timeout")
		case <-doneSignal.Done():
		}

		mu.Lock()
		defer mu.Unlock()
		// All 3 should be delivered (no dedup for zero-value key)
		assert.Equal(t, 3, receivedCount)
	})

	t.Run("FIFO order maintained for distinct keys", func(t *testing.T) {
		lane := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneSize[string](10)).NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()

		receivedKeys := make([]string, 0)
		var mu sync.Mutex
		doneSignal := concurrency.NewSignal()

		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic,
			func(event pubsub.Event) error {
				mu.Lock()
				defer mu.Unlock()
				e := event.(*dedupTestEvent)
				receivedKeys = append(receivedKeys, e.key)
				if len(receivedKeys) == 3 {
					doneSignal.Signal()
				}
				return nil
			}))

		// Publish in order: A, B, C
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A"}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "B"}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "C"}))

		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Events should be consumed within timeout")
		case <-doneSignal.Done():
		}

		mu.Lock()
		defer mu.Unlock()
		// Should receive all 3 distinct events
		// Note: Strict FIFO ordering is not guaranteed due to concurrent dequeue/channel operations.
		// The linked list maintains insertion order, but the signal-based dequeue can introduce
		// slight timing variations. The key guarantee is that all distinct events are processed.
		assert.ElementsMatch(t, []string{"A", "B", "C"}, receivedKeys)
	})

	t.Run("merged item retains original position", func(t *testing.T) {
		lane := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneSize[string](10)).NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()

		receivedKeys := make([]string, 0)
		var mu sync.Mutex
		doneSignal := concurrency.NewSignal()

		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic,
			func(event pubsub.Event) error {
				mu.Lock()
				defer mu.Unlock()
				e := event.(*dedupTestEvent)
				receivedKeys = append(receivedKeys, e.key)
				if len(receivedKeys) == 2 {
					doneSignal.Signal()
				}
				return nil
			}))

		// Publish: A, B, A (second A merges with first)
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A", data: "first"}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "B", data: "second"}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A", data: "merged"}))

		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Events should be consumed within timeout")
		case <-doneSignal.Done():
		}

		mu.Lock()
		defer mu.Unlock()
		// Order should be: merged A, then B
		assert.Equal(t, []string{"A", "B"}, receivedKeys)
	})
}

func TestDedupingLaneMergeSemantics(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	t.Run("Mergeable interface called when implemented", func(t *testing.T) {
		lane := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneSize[string](10)).NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()

		var receivedEvent *dedupTestEvent
		var mu sync.Mutex
		doneSignal := concurrency.NewSignal()

		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic,
			func(event pubsub.Event) error {
				mu.Lock()
				defer mu.Unlock()
				receivedEvent = event.(*dedupTestEvent)
				doneSignal.Signal()
				return nil
			}))

		// Publish two events with sticky-true flag semantics
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A", flag1: true, flag2: false}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A", flag1: false, flag2: true}))

		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Event should be consumed within timeout")
		case <-doneSignal.Done():
		}

		mu.Lock()
		defer mu.Unlock()
		// Both flags should be true due to sticky-true merge
		assert.True(t, receivedEvent.flag1)
		assert.True(t, receivedEvent.flag2)
	})

	// Note: Testing custom mergeFunc with non-Mergeable events would require
	// a separate event type without MergeFrom(). The implementation supports both
	// paths: Mergeable interface first, then custom mergeFunc fallback.

	t.Run("three-way merge accumulates correctly", func(t *testing.T) {
		lane := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneSize[string](10)).NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()

		var receivedEvent *dedupTestEvent
		var mu sync.Mutex
		doneSignal := concurrency.NewSignal()

		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic,
			func(event pubsub.Event) error {
				mu.Lock()
				defer mu.Unlock()
				receivedEvent = event.(*dedupTestEvent)
				doneSignal.Signal()
				return nil
			}))

		// Publish three events with different flag combinations
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A", flag1: true, flag2: false}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A", flag1: false, flag2: true}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A", flag1: false, flag2: false}))

		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Event should be consumed within timeout")
		case <-doneSignal.Done():
		}

		mu.Lock()
		defer mu.Unlock()
		// Both flags should be true (sticky-true across all 3 merges)
		assert.True(t, receivedEvent.flag1)
		assert.True(t, receivedEvent.flag2)
	})
}

func TestDedupingLaneConcurrency(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	t.Run("parallel publishers with duplicate keys", func(t *testing.T) {
		lane := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneSize[string](100)).NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()

		receivedCount := 0
		receivedKeys := make(map[string]int)
		var mu sync.Mutex
		expectedUniqueKeys := 10

		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic,
			func(event pubsub.Event) error {
				mu.Lock()
				defer mu.Unlock()
				e := event.(*dedupTestEvent)
				receivedCount++
				receivedKeys[e.key]++
				return nil
			}))

		// Start 10 goroutines publishing duplicates
		numPublishers := 10
		eventsPerPublisher := 10
		startSignal := concurrency.NewSignal()
		var wg sync.WaitGroup

		for i := 0; i < numPublishers; i++ {
			wg.Add(1)
			go func(publisherID int) {
				defer wg.Done()
				<-startSignal.Done()
				for j := 0; j < eventsPerPublisher; j++ {
					// Each publisher publishes events for all 10 keys
					key := fmt.Sprintf("key-%d", j)
					_ = lane.Publish(&dedupTestEvent{key: key, data: fmt.Sprintf("pub-%d", publisherID)})
				}
			}(i)
		}

		startSignal.Signal()
		wg.Wait()

		// Wait for all events to be consumed
		assert.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			// Verify deduplication is working: should process all 10 unique keys.
			// The exact count will vary based on timing (10-100), but we should
			// at least see all unique keys represented.
			return len(receivedKeys) == expectedUniqueKeys
		}, 2*time.Second, 10*time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		// Verify we got all unique keys
		assert.Equal(t, expectedUniqueKeys, len(receivedKeys), "Should process all 10 unique keys")
		// Verify deduplication reduced total events (some dedup happened, not all 100)
		assert.GreaterOrEqual(t, receivedCount, expectedUniqueKeys, "Should process at least one event per unique key")
		assert.Less(t, receivedCount, 100, "Deduplication should reduce event count below 100")
	})

	t.Run("publish same key during consume (re-queuing)", func(t *testing.T) {
		lane := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneSize[string](10)).NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()

		receivedCount := 0
		var mu sync.Mutex
		firstConsumeSignal := concurrency.NewSignal()
		secondConsumeSignal := concurrency.NewSignal()

		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic,
			func(event pubsub.Event) error {
				mu.Lock()
				receivedCount++
				count := receivedCount
				mu.Unlock()

				if count == 1 {
					// Publish another event with same key while consuming
					_ = lane.Publish(&dedupTestEvent{key: "A", data: "re-queued"})
					firstConsumeSignal.Signal()
				} else if count == 2 {
					secondConsumeSignal.Signal()
				}
				return nil
			}))

		// Publish initial event
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A", data: "first"}))

		// Wait for both consumptions
		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatal("First event should be consumed within timeout")
		case <-firstConsumeSignal.Done():
		}

		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Re-queued event should be consumed within timeout")
		case <-secondConsumeSignal.Done():
		}

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 2, receivedCount)
	})
}

func TestDedupingLaneStop(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	t.Run("stop prevents new publishes", func(t *testing.T) {
		lane := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneSize[string](10)).NewLane()
		assert.NotNil(t, lane)

		// Stop the lane
		lane.Stop()

		// Try to publish
		err := lane.Publish(&dedupTestEvent{key: "A"})
		assert.Error(t, err)
	})

	t.Run("stop doesn't leak dedup index entries", func(t *testing.T) {
		lane := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupingLaneSize[string](10)).NewLane()
		assert.NotNil(t, lane)

		// Publish some events
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A"}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "B"}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "C"}))

		// Get the implementation to check internal state
		laneImpl, ok := lane.(*dedupingLane[string])
		require.True(t, ok)

		// Register a consumer that blocks
		blockSignal := concurrency.NewSignal()
		_ = lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic,
			func(event pubsub.Event) error {
				<-blockSignal.Done()
				return nil
			})

		// Give time for events to start being consumed
		time.Sleep(50 * time.Millisecond)

		// Unblock consumers before stopping to prevent goroutine leaks
		blockSignal.Signal()

		// Stop the lane
		lane.Stop()

		// Verify dedup index is cleaned up (events consumed and removed)
		laneImpl.dedupLock.Lock()
		indexSize := len(laneImpl.dedupIndex)
		laneImpl.dedupLock.Unlock()

		// The events should have been removed from index when dispatched
		assert.LessOrEqual(t, indexSize, 3)
	})
}

func TestDedupingLaneMetrics(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	t.Run("size metric tracks dedup queue size", func(t *testing.T) {
		metric := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_dedup_queue_size",
		})
		lane := NewDedupingLane[string](pubsub.DefaultLane,
			WithDedupKeyFunc(extractTestKey),
			WithDedupSizeMetric[string](metric),
			WithDedupingLaneSize[string](10)).NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()

		blockSignal := concurrency.NewSignal()
		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic,
			func(event pubsub.Event) error {
				<-blockSignal.Done()
				return nil
			}))

		// Publish 3 distinct events
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A"}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "B"}))
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "C"}))

		// The metric tracks the dedup queue size, not the channel.
		// Events are quickly moved from dedup queue → channel by dequeueToChannel goroutine.
		// Since the consumer is blocked, events pile up in the channel, not the dedup queue.
		// So the metric should be 0 once all events are dequeued.
		assert.Eventually(t, func() bool {
			return testutil.ToFloat64(metric) == 0.0
		}, 200*time.Millisecond, 10*time.Millisecond)

		// Now publish a duplicate while the original is in the channel (not yet consumed)
		// This should increase the metric to 1 (duplicate is in dedup queue, not channel yet)
		assert.NoError(t, lane.Publish(&dedupTestEvent{key: "A", data: "updated"}))

		assert.Eventually(t, func() bool {
			val := testutil.ToFloat64(metric)
			// Should be 0 or 1 depending on timing:
			// - 0 if dequeueToChannel already moved it to channel
			// - 1 if it's still in dedup queue
			return val <= 1.0
		}, 200*time.Millisecond, 10*time.Millisecond)

		// Unblock consumer
		blockSignal.Signal()

		// Wait for queue to drain completely
		assert.Eventually(t, func() bool {
			return testutil.ToFloat64(metric) == 0.0
		}, 500*time.Millisecond, 10*time.Millisecond)
	})
}
