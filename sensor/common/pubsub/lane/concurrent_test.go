package lane

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

type concurrentLaneSuite struct {
	suite.Suite
}

func TestConcurrentLane(t *testing.T) {
	suite.Run(t, new(concurrentLaneSuite))
}

func (s *concurrentLaneSuite) TestNewLaneOptions() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("with default options", func() {
		config := NewConcurrentLane(pubsub.DefaultLane)
		assert.Equal(s.T(), pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*concurrentLane)
		require.True(s.T(), ok)
		assert.Equal(s.T(), 0, laneImpl.ch.Cap())
	})
	s.Run("with custom lane size", func() {
		laneSize := 10
		config := NewConcurrentLane(pubsub.DefaultLane, WithConcurrentLaneSize(laneSize))
		assert.Equal(s.T(), pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*concurrentLane)
		require.True(s.T(), ok)
		assert.Equal(s.T(), laneSize, laneImpl.ch.Cap())
	})
	s.Run("with negative lane size", func() {
		laneSize := -1
		config := NewConcurrentLane(pubsub.DefaultLane, WithConcurrentLaneSize(laneSize))
		assert.Equal(s.T(), pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*concurrentLane)
		require.True(s.T(), ok)
		assert.Equal(s.T(), 0, laneImpl.ch.Cap())
	})
	s.Run("with custom consumer", func() {
		config := NewConcurrentLane(pubsub.DefaultLane, WithConcurrentLaneConsumer(newTestConsumer))
		assert.Equal(s.T(), pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*concurrentLane)
		require.True(s.T(), ok)
		assert.NotNil(s.T(), laneImpl.newConsumerFn)
		assert.Len(s.T(), laneImpl.consumerOpts, 0)
	})
	s.Run("with custom consumer and consumer options", func() {
		config := NewConcurrentLane(
			pubsub.DefaultLane,
			WithConcurrentLaneConsumer(newTestConsumer, func(_ pubsub.Consumer) {}),
		)
		assert.Equal(s.T(), pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*concurrentLane)
		require.True(s.T(), ok)
		assert.NotNil(s.T(), laneImpl.newConsumerFn)
		assert.Len(s.T(), laneImpl.consumerOpts, 1)
	})
}

func (s *concurrentLaneSuite) TestOptionPanic() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("panic if WithConcurrentLaneSize is used in a different lane", func() {
		config := &testLaneConfig{
			opts: []pubsub.LaneOption{
				WithConcurrentLaneSize(10),
			},
		}
		s.Assert().Panics(func() {
			config.NewLane()
		})
	})
	s.Run("panic if WithConcurrentLaneConsumer is used in a different lane", func() {
		config := &testLaneConfig{
			opts: []pubsub.LaneOption{
				WithConcurrentLaneConsumer(nil),
			},
		}
		s.Assert().Panics(func() {
			config.NewLane()
		})
	})
	s.Run("panic if a nil NewConsumer is passed to WithConcurrentLaneConsumer", func() {
		config := NewConcurrentLane(pubsub.DefaultLane, WithConcurrentLaneConsumer(nil))
		s.Assert().Panics(func() {
			config.NewLane()
		})
	})
}

func (s *concurrentLaneSuite) TestRegisterConsumer() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("should error on nil callback", func() {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		assert.Error(s.T(), lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, nil))
		lane.Stop()
	})
	s.Run("should successfully register consumer", func() {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		assert.NoError(s.T(), lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			return nil
		}))
		laneImpl, ok := lane.(*concurrentLane)
		require.True(s.T(), ok)
		assert.Len(s.T(), laneImpl.consumers[pubsub.DefaultTopic], 1)
	})
}

func (s *concurrentLaneSuite) TestPublish() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("publish with no consumer should not block", func() {
		lane := NewConcurrentLane(pubsub.DefaultLane, WithConcurrentLaneSize(5)).NewLane()
		assert.NotNil(s.T(), lane)
		assert.NoError(s.T(), lane.Publish(&concurrentTestEvent{}))
		assert.NoError(s.T(), lane.Publish(&concurrentTestEvent{}))
		lane.Stop()
	})
	s.Run("publish and consume concurrently", func() {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		data := "some data"
		consumeSignal := concurrency.NewSignal()
		assert.NoError(s.T(), lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic,
			assertInConcurrentCallback(s.T(), func(t *testing.T, event pubsub.Event) error {
				defer consumeSignal.Signal()
				eventImpl, ok := event.(*concurrentTestEvent)
				require.True(t, ok)
				assert.Equal(t, data, eventImpl.data)
				return nil
			})))
		assert.NoError(s.T(), lane.Publish(&concurrentTestEvent{data: data}))
		select {
		case <-time.After(500 * time.Millisecond):
			s.FailNow("Event should be consumed within timeout")
		case <-consumeSignal.Done():
		}
		lane.Stop()
	})
	s.Run("publish should not block with slow consumer (concurrent processing)", func() {
		lane := NewConcurrentLane(pubsub.DefaultLane, WithConcurrentLaneSize(5)).NewLane()
		assert.NotNil(s.T(), lane)
		unblockSig := concurrency.NewSignal()
		var processedCount atomic.Int32
		processedCount.Store(0)
		assert.NoError(s.T(), lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			defer processedCount.Add(1)
			<-unblockSig.Done()
			return nil
		}))
		numEvents := 3
		// Publish multiple events - they should not block even though consumer is slow
		for i := 0; i < numEvents; i++ {
			require.NoError(s.T(), lane.Publish(&concurrentTestEvent{data: "event"}))
		}
		unblockSig.Signal()
		assert.Eventually(s.T(), func() bool {
			return processedCount.Load() == int32(numEvents)
		}, 100*time.Millisecond, 10*time.Millisecond)
		lane.Stop()
	})
	s.Run("stop should prevent new publishes", func() {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		assert.NoError(s.T(), lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			return nil
		}))
		lane.Stop()
		assert.Error(s.T(), lane.Publish(&concurrentTestEvent{}))
	})
	s.Run("multiple consumers receive same event", func() {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		consumer1Signal := concurrency.NewSignal()
		consumer2Signal := concurrency.NewSignal()
		assert.NoError(s.T(), lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			consumer1Signal.Signal()
			return nil
		}))
		assert.NoError(s.T(), lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			consumer2Signal.Signal()
			return nil
		}))
		assert.NoError(s.T(), lane.Publish(&concurrentTestEvent{data: "broadcast"}))
		select {
		case <-time.After(500 * time.Millisecond):
			s.FailNow("First consumer should receive event within timeout")
		case <-consumer1Signal.Done():
		}
		select {
		case <-time.After(500 * time.Millisecond):
			s.FailNow("Second consumer should receive event within timeout")
		case <-consumer2Signal.Done():
		}
		lane.Stop()
	})
}

func (s *concurrentLaneSuite) TestErrorHandling() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("consumer error should be logged", func() {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		expectedErr := errors.New("consumer error")
		errorReturned := concurrency.NewSignal()
		assert.NoError(s.T(), lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			defer errorReturned.Signal()
			return expectedErr
		}))
		initialCounter := testutil.ToFloat64(metrics.GetConsumerOperationMetric().WithLabelValues(
			pubsub.DefaultLane.String(),
			pubsub.DefaultTopic.String(),
			pubsub.DefaultConsumer.String(),
			metrics.ConsumerError.String()))
		assert.NoError(s.T(), lane.Publish(&concurrentTestEvent{data: "error event"}))
		// Give time for error to be handled
		select {
		case <-time.After(500 * time.Millisecond):
			s.FailNow("Error should be returned within timeout")
		case <-errorReturned.Done():
		}
		assert.Eventually(s.T(), func() bool {
			counter := metrics.GetConsumerOperationMetric().WithLabelValues(
				pubsub.DefaultLane.String(),
				pubsub.DefaultTopic.String(),
				pubsub.DefaultConsumer.String(),
				metrics.ConsumerError.String())
			return testutil.ToFloat64(counter) > initialCounter
		}, 100*time.Millisecond, 10*time.Millisecond)
		lane.Stop()
	})
	s.Run("publish to topic with no consumers should log error", func() {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		unknownTopic := pubsub.Topic(999)
		initialCounter := testutil.ToFloat64(metrics.GetConsumerOperationMetric().WithLabelValues(
			pubsub.DefaultLane.String(),
			unknownTopic.String(),
			pubsub.NoConsumers.String(),
			metrics.ConsumerError.String()))
		// Publish to topic with no consumers
		assert.NoError(s.T(), lane.Publish(&concurrentTestEvent{customTopic: &unknownTopic}))
		assert.Eventually(s.T(), func() bool {
			counter := metrics.GetConsumerOperationMetric().WithLabelValues(
				pubsub.DefaultLane.String(),
				unknownTopic.String(),
				pubsub.NoConsumers.String(),
				metrics.NoConsumers.String())
			return testutil.ToFloat64(counter) > initialCounter
		}, 100*time.Millisecond, 10*time.Millisecond)
		lane.Stop()
	})
}

func (s *concurrentLaneSuite) TestStop() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("stop should clean up resources", func() {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		lane.Stop()
		laneImpl, ok := lane.(*concurrentLane)
		require.True(s.T(), ok)

		_, ok = <-laneImpl.ch.Chan()
		assert.False(s.T(), ok)
		_, ok = <-laneImpl.errC.Chan()
		assert.False(s.T(), ok)
	})
	s.Run("stop should stop all consumers", func() {
		lane := NewConcurrentLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		consumerStopped := false
		mockConsumer := &mockConsumer{
			stopFn: func() {
				consumerStopped = true
			},
		}
		laneImpl := lane.(*concurrentLane)
		laneImpl.consumers[pubsub.DefaultTopic] = []pubsub.Consumer{mockConsumer}
		lane.Stop()
		assert.True(s.T(), consumerStopped)
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
