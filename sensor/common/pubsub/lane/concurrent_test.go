package lane

import (
	"errors"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewLaneOptions(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)
	t.Run("with default options", func(t *testing.T) {
		config := NewConcurrentLane(pubsub.DefaultLane)
		assert.Equal(t, pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*concurrentLane)
		require.True(t, ok)
		assert.Equal(t, 0, laneImpl.ch.Cap())
	})
	t.Run("with custom lane size", func(t *testing.T) {
		laneSize := 10
		config := NewConcurrentLane(pubsub.DefaultLane, WithConcurrentLaneSize(laneSize))
		assert.Equal(t, pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*concurrentLane)
		require.True(t, ok)
		assert.Equal(t, laneSize, laneImpl.ch.Cap())
	})
	t.Run("with negative lane size", func(t *testing.T) {
		laneSize := -1
		config := NewConcurrentLane(pubsub.DefaultLane, WithConcurrentLaneSize(laneSize))
		assert.Equal(t, pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*concurrentLane)
		require.True(t, ok)
		assert.Equal(t, 0, laneImpl.ch.Cap())
	})
	t.Run("with custom consumer", func(t *testing.T) {
		config := NewConcurrentLane(pubsub.DefaultLane, WithConcurrentLaneConsumer(newTestConsumer))
		assert.Equal(t, pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*concurrentLane)
		require.True(t, ok)
		assert.NotNil(t, laneImpl.newConsumerFn)
		assert.Len(t, laneImpl.consumerOpts, 0)
	})
	t.Run("with custom consumer and consumer options", func(t *testing.T) {
		config := NewConcurrentLane(
			pubsub.DefaultLane,
			WithConcurrentLaneConsumer(newTestConsumer, func(_ pubsub.Consumer) {}),
		)
		assert.Equal(t, pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*concurrentLane)
		require.True(t, ok)
		assert.NotNil(t, laneImpl.newConsumerFn)
		assert.Len(t, laneImpl.consumerOpts, 1)
	})
}

func TestOptionPanic(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)
	t.Run("panic if WithConcurrentLaneSize is used in a different lane", func(t *testing.T) {
		config := &testLaneConfig{
			opts: []pubsub.LaneOption{
				WithConcurrentLaneSize(10),
			},
		}
		assert.Panics(t, func() {
			config.NewLane()
		})
	})
	t.Run("panic if WithConcurrentLaneConsumer is used in a different lane", func(t *testing.T) {
		config := &testLaneConfig{
			opts: []pubsub.LaneOption{
				WithConcurrentLaneConsumer(nil),
			},
		}
		assert.Panics(t, func() {
			config.NewLane()
		})
	})
	t.Run("panic if a nil NewConsumer is passed to WithConcurrentLaneConsumer", func(t *testing.T) {
		config := NewConcurrentLane(pubsub.DefaultLane, WithConcurrentLaneConsumer(nil))
		assert.Panics(t, func() {
			config.NewLane()
		})
	})
}

func TestRegisterConsumer(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)
	t.Run("should error on nil callback", func(t *testing.T) {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(t, lane)
		assert.Error(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, nil))
		lane.Stop()
	})
	t.Run("should successfully register consumer", func(t *testing.T) {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(t, lane)
		defer lane.Stop()
		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			return nil
		}))
		laneImpl, ok := lane.(*concurrentLane)
		require.True(t, ok)
		assert.Len(t, laneImpl.consumers[pubsub.DefaultTopic], 1)
	})
}

func TestPublish(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)
	t.Run("publish with no consumer should not block", func(t *testing.T) {
		lane := NewConcurrentLane(pubsub.DefaultLane, WithConcurrentLaneSize(5)).NewLane()
		assert.NotNil(t, lane)
		assert.NoError(t, lane.Publish(&concurrentTestEvent{}))
		assert.NoError(t, lane.Publish(&concurrentTestEvent{}))
		lane.Stop()
	})
	t.Run("publish and consume concurrently", func(t *testing.T) {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(t, lane)
		data := "some data"
		consumeSignal := concurrency.NewSignal()
		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic,
			assertInConcurrentCallback(t, func(t *testing.T, event pubsub.Event) error {
				defer consumeSignal.Signal()
				eventImpl, ok := event.(*concurrentTestEvent)
				require.True(t, ok)
				assert.Equal(t, data, eventImpl.data)
				return nil
			})))
		assert.NoError(t, lane.Publish(&concurrentTestEvent{data: data}))
		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Event should be consumed within timeout")
		case <-consumeSignal.Done():
		}
		lane.Stop()
	})
	t.Run("publish should not block with slow consumer (concurrent processing)", func(t *testing.T) {
		lane := NewConcurrentLane(pubsub.DefaultLane, WithConcurrentLaneSize(5)).NewLane()
		assert.NotNil(t, lane)
		unblockSig := concurrency.NewSignal()
		numEvents := 3
		doneCh := make(chan struct{}, numEvents)
		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			<-unblockSig.Done()
			doneCh <- struct{}{}
			return nil
		}))
		// Publish multiple events - they should not block even though consumer is slow
		for i := 0; i < numEvents; i++ {
			require.NoError(t, lane.Publish(&concurrentTestEvent{data: "event"}))
		}
		unblockSig.Signal()
		// Wait for all events to be processed
		for i := 0; i < numEvents; i++ {
			select {
			case <-doneCh:
			case <-time.After(1 * time.Second):
				t.Fatalf("Event %d was not processed within timeout", i)
			}
		}
		lane.Stop()
	})
	t.Run("stop should prevent new publishes", func(t *testing.T) {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(t, lane)
		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			return nil
		}))
		lane.Stop()
		assert.Error(t, lane.Publish(&concurrentTestEvent{}))
	})
	t.Run("multiple consumers receive same event", func(t *testing.T) {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(t, lane)
		consumer1Signal := concurrency.NewSignal()
		consumer2Signal := concurrency.NewSignal()
		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			consumer1Signal.Signal()
			return nil
		}))
		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			consumer2Signal.Signal()
			return nil
		}))
		assert.NoError(t, lane.Publish(&concurrentTestEvent{data: "broadcast"}))
		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatal("First consumer should receive event within timeout")
		case <-consumer1Signal.Done():
		}
		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Second consumer should receive event within timeout")
		case <-consumer2Signal.Done():
		}
		lane.Stop()
	})
}

func TestErrorHandling(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)
	t.Run("consumer error should be logged", func(t *testing.T) {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(t, lane)
		expectedErr := errors.New("consumer error")
		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			return expectedErr
		}))
		initialCounter := testutil.ToFloat64(metrics.GetConsumerOperationMetric().WithLabelValues(
			pubsub.DefaultLane.String(),
			pubsub.DefaultTopic.String(),
			pubsub.DefaultConsumer.String(),
			metrics.ConsumerError.String()))
		assert.NoError(t, lane.Publish(&concurrentTestEvent{data: "error event"}))
		assert.Eventually(t, func() bool {
			counter := metrics.GetConsumerOperationMetric().WithLabelValues(
				pubsub.DefaultLane.String(),
				pubsub.DefaultTopic.String(),
				pubsub.DefaultConsumer.String(),
				metrics.ConsumerError.String())
			return testutil.ToFloat64(counter) > initialCounter
		}, 100*time.Millisecond, 10*time.Millisecond)
		lane.Stop()
	})
	t.Run("publish to topic with no consumers should log error", func(t *testing.T) {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(t, lane)
		unknownTopic := pubsub.Topic(999)

		// Register a barrier consumer on a different topic to ensure event processing completes
		barrierTopic := pubsub.Topic(998)
		barrierDone := make(chan struct{})
		assert.NoError(t, lane.RegisterConsumer(pubsub.DefaultConsumer, barrierTopic, func(_ pubsub.Event) error {
			close(barrierDone)
			return nil
		}))

		initialCounter := testutil.ToFloat64(metrics.GetConsumerOperationMetric().WithLabelValues(
			pubsub.DefaultLane.String(),
			unknownTopic.String(),
			pubsub.NoConsumers.String(),
			metrics.NoConsumers.String()))

		// Publish to topic with no consumers
		assert.NoError(t, lane.Publish(&concurrentTestEvent{customTopic: &unknownTopic}))

		// Publish barrier event - when this completes, we know the previous event was processed
		assert.NoError(t, lane.Publish(&concurrentTestEvent{customTopic: &barrierTopic}))
		select {
		case <-time.After(1 * time.Second):
			t.Fatal("Barrier event not processed within timeout")
		case <-barrierDone:
		}

		// Verify metrics were updated
		counter := metrics.GetConsumerOperationMetric().WithLabelValues(
			pubsub.DefaultLane.String(),
			unknownTopic.String(),
			pubsub.NoConsumers.String(),
			metrics.NoConsumers.String())
		assert.Greater(t, testutil.ToFloat64(counter), initialCounter)
		lane.Stop()
	})
}

func TestStop(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)
	t.Run("stop should clean up resources", func(t *testing.T) {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(t, lane)
		lane.Stop()
		laneImpl, ok := lane.(*concurrentLane)
		require.True(t, ok)

		_, ok = <-laneImpl.ch.Chan()
		assert.False(t, ok)
	})
	t.Run("stop should stop all consumers", func(t *testing.T) {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(t, lane)
		consumerStopped := false
		mockConsumer := &mockConsumer{
			stopFn: func() {
				consumerStopped = true
			},
		}
		laneImpl := lane.(*concurrentLane)
		laneImpl.consumers[pubsub.DefaultTopic] = []pubsub.Consumer{mockConsumer}
		lane.Stop()
		assert.True(t, consumerStopped)
	})
}

func assertInConcurrentCallback(t *testing.T, assertion func(*testing.T, pubsub.Event) error) pubsub.EventCallback {
	return func(event pubsub.Event) error {
		return assertion(t, event)
	}
}

// testEvent with topic field for flexibility in tests
type concurrentTestEvent struct {
	data        string
	customTopic *pubsub.Topic
}

func (t *concurrentTestEvent) Topic() pubsub.Topic {
	if t.customTopic != nil {
		return *t.customTopic
	}
	return pubsub.DefaultTopic
}

func (t *concurrentTestEvent) Lane() pubsub.LaneID {
	return pubsub.DefaultLane
}

type mockConsumer struct {
	stopFn func()
}

func (m *mockConsumer) Consume(_ concurrency.Waitable, _ pubsub.Event) <-chan error {
	errC := make(chan error)
	close(errC)
	return errC
}

func (m *mockConsumer) Stop() {
	if m.stopFn != nil {
		m.stopFn()
	}
}
