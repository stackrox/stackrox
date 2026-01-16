package consumer

import (
	"errors"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type bufferedConsumerSuite struct {
	suite.Suite
}

func TestBufferedConsumer(t *testing.T) {
	suite.Run(t, new(bufferedConsumerSuite))
}

func (s *bufferedConsumerSuite) TestNewBufferedConsumer() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("should error on nil callback", func() {
		consumer, err := NewBufferedConsumer(nil)
		assert.Nil(s.T(), consumer)
		assert.Error(s.T(), err)
	})
	s.Run("with default options", func() {
		consumer, err := NewBufferedConsumer(func(_ pubsub.Event) error { return nil })
		require.NoError(s.T(), err)
		assert.NotNil(s.T(), consumer)
		defer consumer.Stop()
		consumerImpl, ok := consumer.(*BufferedConsumer)
		require.True(s.T(), ok)
		assert.Equal(s.T(), 10, consumerImpl.size)
		assert.Equal(s.T(), 10, cap(consumerImpl.buffer))
	})
	s.Run("with custom buffer size", func() {
		bufferSize := 20
		consumer, err := NewBufferedConsumer(
			func(_ pubsub.Event) error { return nil },
			WithBufferedConsumerSize(bufferSize),
		)
		require.NoError(s.T(), err)
		assert.NotNil(s.T(), consumer)
		defer consumer.Stop()
		consumerImpl, ok := consumer.(*BufferedConsumer)
		require.True(s.T(), ok)
		assert.Equal(s.T(), bufferSize, consumerImpl.size)
		assert.Equal(s.T(), bufferSize, cap(consumerImpl.buffer))
	})
	s.Run("with negative buffer size", func() {
		consumer, err := NewBufferedConsumer(
			func(_ pubsub.Event) error { return nil },
			WithBufferedConsumerSize(-1),
		)
		require.NoError(s.T(), err)
		assert.NotNil(s.T(), consumer)
		defer consumer.Stop()
		consumerImpl, ok := consumer.(*BufferedConsumer)
		require.True(s.T(), ok)
		// Negative size should be ignored and default should be used
		assert.Equal(s.T(), 10, consumerImpl.size)
		assert.Equal(s.T(), 10, cap(consumerImpl.buffer))
	})
}

func (s *bufferedConsumerSuite) TestConsume() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("consume single event successfully", func() {
		processed := concurrency.NewSignal()
		var receivedEvent pubsub.Event
		consumer, err := NewBufferedConsumer(func(event pubsub.Event) error {
			defer processed.Signal()
			receivedEvent = event
			return nil
		})
		require.NoError(s.T(), err)
		defer consumer.Stop()

		event := &testEvent{data: "test data"}
		waitable := concurrency.NewSignal()
		errC := consumer.Consume(&waitable, event)

		select {
		case <-time.After(100 * time.Millisecond):
			s.FailNow("Event should be processed within timeout")
		case <-processed.Done():
		}

		select {
		case err := <-errC:
			assert.NoError(s.T(), err)
		case <-time.After(100 * time.Millisecond):
			s.FailNow("Error channel should be closed")
		}

		assert.Equal(s.T(), event, receivedEvent)
	})

	s.Run("consume event with callback error", func() {
		expectedErr := errors.New("callback error")
		consumer, err := NewBufferedConsumer(func(_ pubsub.Event) error {
			return expectedErr
		})
		require.NoError(s.T(), err)
		defer consumer.Stop()

		event := &testEvent{data: "test data"}
		waitable := concurrency.NewSignal()
		errC := consumer.Consume(&waitable, event)

		select {
		case err := <-errC:
			assert.Equal(s.T(), expectedErr, err)
		case <-time.After(100 * time.Millisecond):
			s.FailNow("Should receive callback error")
		}
	})

	s.Run("consume with buffer full", func() {
		blockSignal := concurrency.NewSignal()
		processStarted := concurrency.NewSignal()
		once := sync.Once{}
		consumer, err := NewBufferedConsumer(
			func(_ pubsub.Event) error {
				once.Do(func() {
					processStarted.Signal()
				})
				<-blockSignal.Done()
				return nil
			},
			WithBufferedConsumerSize(1),
		)
		require.NoError(s.T(), err)
		consumerImpl, ok := consumer.(*BufferedConsumer)
		require.True(s.T(), ok)
		defer consumer.Stop()

		waitable := concurrency.NewSignal()
		// First consume puts event in buffer
		errC1 := consumer.Consume(&waitable, &testEvent{data: "event 1"})

		// Wait for processing to start
		select {
		case <-processStarted.Done():
		case <-time.After(100 * time.Millisecond):
			s.FailNow("Processing should have started")
		}

		// Second consume should succeed and fill the buffer
		errC2 := consumer.Consume(&waitable, &testEvent{data: "event 2"})
		// Wait until the event 2 is put in the channel
		assert.Eventually(s.T(), func() bool {
			return len(consumerImpl.buffer) == 1
		}, 100*time.Millisecond, 10*time.Millisecond)

		// Third consume should fail because buffer is full
		errC3 := consumer.Consume(&waitable, &testEvent{data: "event 3"})
		select {
		case err := <-errC3:
			assert.Error(s.T(), err)
			assert.Contains(s.T(), err.Error(), "buffer is full")
		case <-time.After(100 * time.Millisecond):
			s.FailNow("Should receive buffer full error")
		}

		blockSignal.Signal()

		// Wait for events to complete
		select {
		case err := <-errC1:
			assert.NoError(s.T(), err)
		case <-time.After(200 * time.Millisecond):
			s.FailNow("Should finish processing event 1")
		}
		select {
		case err := <-errC2:
			assert.NoError(s.T(), err)
		case <-time.After(200 * time.Millisecond):
			s.FailNow("Should finish processing event 2")
		}
	})

	s.Run("consume should stop on consumer stopped", func() {
		consumer, err := NewBufferedConsumer(func(_ pubsub.Event) error {
			s.FailNow("should not process the event")
			return nil
		})
		require.NoError(s.T(), err)
		consumerImpl, ok := consumer.(*BufferedConsumer)
		require.True(s.T(), ok)

		consumer.Stop()
		assert.Eventually(s.T(), func() bool {
			return consumerImpl.buffer == nil
		}, 100*time.Millisecond, 10*time.Millisecond)

		waitable := concurrency.NewSignal()

		errC := consumer.Consume(&waitable, &testEvent{data: "test data"})

		select {
		case _ = <-errC:
			// error or nil are both acceptable because the select chooses a
			// random case path when the stop signal and the writing in the
			// channel are both eligible at the same time.
		case <-time.After(100 * time.Millisecond):
			s.FailNow("timeout waiting for Consume to finish")
		}
	})
	s.Run("consume should stop on waitable", func() {
		blockSignal := concurrency.NewSignal()
		consumer, err := NewBufferedConsumer(func(_ pubsub.Event) error {
			<-blockSignal.Done()
			return nil
		})
		require.NoError(s.T(), err)

		defer consumer.Stop()

		waitable := concurrency.NewSignal()
		waitable.Signal()
		<-waitable.Done()

		errC := consumer.Consume(&waitable, &testEvent{data: "test data"})

		select {
		case err := <-errC:
			assert.NoError(s.T(), err)
		case <-time.After(100 * time.Millisecond):
			s.FailNow("timeout waiting for Consume to finish")
		}
		blockSignal.Signal()
	})
}

func (s *bufferedConsumerSuite) TestStop() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("stop should close buffer and stop processing", func() {
		consumer, err := NewBufferedConsumer(func(_ pubsub.Event) error {
			return nil
		})
		require.NoError(s.T(), err)

		consumer.Stop()

		consumerImpl := consumer.(*BufferedConsumer)
		assert.Nil(s.T(), consumerImpl.buffer)
	})
}

func (s *bufferedConsumerSuite) TestConcurrentConsume() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("multiple concurrent consume calls", func() {
		processedCount := 0
		var mu sync.Mutex
		consumer, err := NewBufferedConsumer(func(_ pubsub.Event) error {
			mu.Lock()
			defer mu.Unlock()
			processedCount++
			return nil
		}, WithBufferedConsumerSize(20))
		require.NoError(s.T(), err)
		defer consumer.Stop()

		waitable := concurrency.NewSignal()
		numEvents := 10
		errChannels := make([]<-chan error, numEvents)

		for i := 0; i < numEvents; i++ {
			errChannels[i] = consumer.Consume(&waitable, &testEvent{data: "event"})
		}

		// Wait for all to complete
		for _, errC := range errChannels {
			select {
			case err := <-errC:
				assert.NoError(s.T(), err)
			case <-time.After(500 * time.Millisecond):
				s.FailNow("Events should be processed within timeout")
			}
		}

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(s.T(), numEvents, processedCount)
	})
}
