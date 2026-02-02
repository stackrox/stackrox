package consumer

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/channel"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBufferedConsumer_NilCallback(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	c, err := NewBufferedConsumer(pubsub.DefaultLane, pubsub.DefaultTopic, pubsub.DefaultConsumer, nil)
	assert.Error(t, err)
	assert.Nil(t, c)
}

func TestBufferedConsumer_ConsumeSuccess(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		callbackCalled := false
		eventData := "test-data"

		c, err := NewBufferedConsumer(pubsub.DefaultLane, pubsub.DefaultTopic, pubsub.DefaultConsumer, func(event pubsub.Event) error {
			callbackCalled = true
			te, ok := event.(*testEvent)
			require.True(t, ok)
			assert.Equal(t, eventData, te.data)
			return nil
		})
		require.NoError(t, err)
		defer c.Stop()

		ctx := context.Background()
		errC := c.Consume(ctx, &testEvent{data: eventData})

		// Wait for callback to complete
		synctest.Wait()
		assert.True(t, callbackCalled)

		// Verify errC receives nil and closes
		err, ok := <-errC
		assert.True(t, ok, "errC should be open for first read")
		assert.Nil(t, err)

		err, ok = <-errC
		assert.False(t, ok, "errC should be closed after first read")
		assert.Nil(t, err)
	})
}

func TestBufferedConsumer_CallbackError(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		expectedErr := errors.New("callback error")

		c, err := NewBufferedConsumer(pubsub.DefaultLane, pubsub.DefaultTopic, pubsub.DefaultConsumer, func(event pubsub.Event) error {
			return expectedErr
		})
		require.NoError(t, err)
		defer c.Stop()

		ctx := context.Background()
		errC := c.Consume(ctx, &testEvent{data: "test"})

		// Wait for callback to complete
		synctest.Wait()

		// Verify errC receives the callback error and closes
		err, ok := <-errC
		assert.True(t, ok, "errC should be open for error")
		assert.Equal(t, expectedErr, err)

		err, ok = <-errC
		assert.False(t, ok, "errC should be closed after error")
		assert.Nil(t, err)
	})
}

func TestBufferedConsumer_BufferFull(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		blockCallback := concurrency.NewSignal()

		c, err := NewBufferedConsumer(
			pubsub.DefaultLane,
			pubsub.DefaultTopic,
			pubsub.DefaultConsumer,
			func(event pubsub.Event) error {
				<-blockCallback.Done()
				return nil
			},
			WithBufferedConsumerSize(1),
		)
		require.NoError(t, err)
		defer c.Stop()

		ctx := context.Background()

		// Block the buffer
		errC1 := c.Consume(ctx, &testEvent{data: "1"})
		synctest.Wait()

		// This event should be buffered
		errC2 := c.Consume(ctx, &testEvent{data: "2"})
		synctest.Wait()

		// Buffer should be full - this should fail with ErrChannelFull
		errC3 := c.Consume(ctx, &testEvent{data: "3"})
		synctest.Wait()

		// Second consume should get buffer full error
		err, ok := <-errC3
		assert.True(t, ok)
		assert.Equal(t, channel.ErrChannelFull, err)

		err, ok = <-errC3
		assert.False(t, ok, "errC2 should be closed")
		assert.Nil(t, err)

		// First errC should still be open (callback blocked)
		select {
		case <-errC1:
			t.Fatal("errC1 should still be open, callback is blocked")
		default:
		}
		// Second errC should still be open (callback blocked)
		select {
		case <-errC2:
			t.Fatal("errC2 should still be open, callback is blocked")
		default:
		}

		// Unblock callback and wait for completion
		blockCallback.Signal()
		synctest.Wait()

		// Now errC1 should complete
		err, ok = <-errC1
		assert.True(t, ok)
		assert.Nil(t, err)

		err, ok = <-errC1
		assert.False(t, ok)
		assert.Nil(t, err)

		// Now errC2 should complete
		err, ok = <-errC2
		assert.True(t, ok)
		assert.Nil(t, err)

		err, ok = <-errC2
		assert.False(t, ok)
		assert.Nil(t, err)
	})
}

func TestBufferedConsumer_WaitableCancellation(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		c, err := NewBufferedConsumer(pubsub.DefaultLane, pubsub.DefaultTopic, pubsub.DefaultConsumer, func(event pubsub.Event) error {
			return nil
		})
		require.NoError(t, err)
		defer c.Stop()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel before Consume

		errC := c.Consume(ctx, &testEvent{data: "test"})
		synctest.Wait()

		// errC should close immediately without processing
		err, ok := <-errC
		assert.False(t, ok, "errC should be closed when waitable already cancelled")
		assert.Nil(t, err)
	})
}

func TestBufferedConsumer_StopDuringConsume(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		blockCallback := make(chan struct{})

		c, err := NewBufferedConsumer(pubsub.DefaultLane, pubsub.DefaultTopic, pubsub.DefaultConsumer, func(event pubsub.Event) error {
			<-blockCallback
			return nil
		})
		require.NoError(t, err)

		ctx := context.Background()
		errC := c.Consume(ctx, &testEvent{data: "test"})

		// Wait for event to be buffered and callback to start
		synctest.Wait()

		// Stop the consumer (callback is still blocked)
		c.Stop()
		close(blockCallback)

		// errC should close without sending error
		err, ok := <-errC
		assert.False(t, ok, "errC should be closed after Stop")
		assert.Nil(t, err)
	})
}

func TestBufferedConsumer_ConcurrentConsume(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		const numEvents = 10
		var callbackCount atomic.Int32

		c, err := NewBufferedConsumer(pubsub.DefaultLane, pubsub.DefaultTopic, pubsub.DefaultConsumer, func(event pubsub.Event) error {
			callbackCount.Add(1)
			return nil
		})
		require.NoError(t, err)
		defer c.Stop()

		ctx := context.Background()
		var errChannelsLock sync.Mutex
		var errChannels []<-chan error

		// Consume multiple events concurrently
		for range numEvents {
			go func() {
				errC := c.Consume(ctx, &testEvent{data: "test"})
				concurrency.WithLock(&errChannelsLock, func() {
					errChannels = append(errChannels, errC)
				})
			}()
		}

		// Wait for all to complete
		synctest.Wait()

		// All callbacks should have been called
		assert.Equal(t, int32(numEvents), callbackCount.Load())

		// All errC channels should complete successfully
		for i, errC := range errChannels {
			err, ok := <-errC
			assert.True(t, ok, "errC %d should be open", i)
			assert.Nil(t, err, "errC %d should have no error", i)

			err, ok = <-errC
			assert.False(t, ok, "errC %d should be closed", i)
			assert.Nil(t, err)
		}
	})
}

func TestBufferedConsumer_WithBufferedConsumerSize(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		c, err := NewBufferedConsumer(
			pubsub.DefaultLane,
			pubsub.DefaultTopic,
			pubsub.DefaultConsumer,
			func(event pubsub.Event) error { return nil },
			WithBufferedConsumerSize(5),
		)
		require.NoError(t, err)
		defer c.Stop()

		impl, ok := c.(*BufferedConsumer)
		require.True(t, ok)
		assert.Equal(t, 5, impl.size)
		assert.Equal(t, 5, impl.buffer.Cap())
	})
}

func TestBufferedConsumer_WithBufferedConsumerSize_Negative(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		// Negative size should be ignored
		c, err := NewBufferedConsumer(
			pubsub.DefaultLane,
			pubsub.DefaultTopic,
			pubsub.DefaultConsumer,
			func(event pubsub.Event) error { return nil },
			WithBufferedConsumerSize(-1),
		)
		require.NoError(t, err)
		defer c.Stop()

		impl, ok := c.(*BufferedConsumer)
		require.True(t, ok)
		assert.Equal(t, 1000, impl.size, "should use default size")
	})
}

func TestBufferedConsumer_StopIdempotent(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		c, err := NewBufferedConsumer(pubsub.DefaultLane, pubsub.DefaultTopic, pubsub.DefaultConsumer, func(event pubsub.Event) error {
			return nil
		})
		require.NoError(t, err)

		// Multiple Stop() calls should not panic
		c.Stop()
		c.Stop()
		c.Stop()
	})
}

func TestBufferedConsumer_ConsumeAfterStop(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		c, err := NewBufferedConsumer(pubsub.DefaultLane, pubsub.DefaultTopic, pubsub.DefaultConsumer, func(event pubsub.Event) error {
			return nil
		})
		require.NoError(t, err)

		c.Stop()
		synctest.Wait()

		ctx := context.Background()
		errC := c.Consume(ctx, &testEvent{data: "test"})
		synctest.Wait()

		// errC should be closed immediately without error since stopper is triggered
		err, ok := <-errC
		assert.False(t, ok, "errC should be closed when consumer is stopped")
		assert.Nil(t, err)
	})
}
