package safe

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeChannel_Write_Success(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := NewChannel[int](5, ctx)

	// Write some items
	err := ch.Write(1)
	require.NoError(t, err)

	err = ch.Write(2)
	require.NoError(t, err)

	err = ch.Write(3)
	require.NoError(t, err)

	// Read and verify
	assert.Equal(t, 1, <-ch.Chan())
	assert.Equal(t, 2, <-ch.Chan())
	assert.Equal(t, 3, <-ch.Chan())
}

func TestSafeChannel_Write_BlocksWhenFull(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch := NewChannel[int](2, ctx)

		// Fill the channel
		require.NoError(t, ch.Write(1))
		require.NoError(t, ch.Write(2))

		// Write should block
		writeStarted := concurrency.NewSignal()
		writeCompleted := concurrency.NewSignal()
		go func() {
			writeStarted.Signal()
			_ = ch.Write(3)
			writeCompleted.Signal()
		}()

		// Wait for write to start
		<-writeStarted.Done()

		// Wait for the goroutine to become blocked
		synctest.Wait()

		// Verify write has not completed (still blocked)
		select {
		case <-writeCompleted.Done():
			t.Fatal("Write should have blocked on full channel")
		default:
			// Expected - write is blocked
		}

		// Unblock by reading
		assert.Equal(t, 1, <-ch.Chan())

		// Wait for write to complete
		<-writeCompleted.Done()

		// Verify the third item was written
		assert.Equal(t, 2, <-ch.Chan())
		assert.Equal(t, 3, <-ch.Chan())
	})
}

func TestSafeChannel_Write_FailsAfterWaitableTriggered(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	ctx, cancel := context.WithCancel(context.Background())
	ch := NewChannel[int](5, ctx)

	// Cancel the context
	cancel()

	// Write should fail
	err := ch.Write(1)
	assert.ErrorIs(t, err, ErrWaitableTriggered)
}

func TestSafeChannel_TryWrite_Success(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := NewChannel[int](5, ctx)

	// TryWrite some items
	err := ch.TryWrite(1)
	require.NoError(t, err)

	err = ch.TryWrite(2)
	require.NoError(t, err)

	// Read and verify
	assert.Equal(t, 1, <-ch.Chan())
	assert.Equal(t, 2, <-ch.Chan())
}

func TestSafeChannel_TryWrite_FailsWhenFull(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := NewChannel[int](2, ctx)

	// Fill the channel
	require.NoError(t, ch.TryWrite(1))
	require.NoError(t, ch.TryWrite(2))

	// TryWrite should fail immediately
	err := ch.TryWrite(3)
	assert.ErrorIs(t, err, ErrChannelFull)

	// Channel should still have the original items
	assert.Equal(t, 1, <-ch.Chan())
	assert.Equal(t, 2, <-ch.Chan())
}

func TestSafeChannel_TryWrite_FailsAfterWaitableTriggered(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	ctx, cancel := context.WithCancel(context.Background())
	ch := NewChannel[int](5, ctx)

	// Cancel the context
	cancel()

	// TryWrite should fail
	err := ch.TryWrite(1)
	assert.ErrorIs(t, err, ErrWaitableTriggered)
}

func TestSafeChannel_Len(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := NewChannel[int](5, ctx)

	assert.Equal(t, 0, ch.Len())

	require.NoError(t, ch.Write(1))
	assert.Equal(t, 1, ch.Len())

	require.NoError(t, ch.Write(2))
	assert.Equal(t, 2, ch.Len())

	<-ch.Chan()
	assert.Equal(t, 1, ch.Len())

	<-ch.Chan()
	assert.Equal(t, 0, ch.Len())
}

func TestSafeChannel_Cap(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := NewChannel[int](10, ctx)
	assert.Equal(t, 10, ch.Cap())

	// Capacity doesn't change as we write
	require.NoError(t, ch.Write(1))
	assert.Equal(t, 10, ch.Cap())

	require.NoError(t, ch.Write(2))
	assert.Equal(t, 10, ch.Cap())

	// Capacity doesn't change as we read
	<-ch.Chan()
	assert.Equal(t, 10, ch.Cap())
}

func TestSafeChannel_NegativeSize(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Negative size should be treated as 0 (unbuffered)
		ch := NewChannel[int](-5, ctx)
		assert.Equal(t, 0, ch.Cap())
		assert.Equal(t, 0, ch.Len())

		// Should still work as an unbuffered channel
		writeStarted := concurrency.NewSignal()
		writeCompleted := concurrency.NewSignal()
		go func() {
			writeStarted.Signal()
			require.NoError(t, ch.Write(42))
			writeCompleted.Signal()
		}()

		// Wait for write to start
		<-writeStarted.Done()

		// Wait for goroutine to become blocked
		synctest.Wait()

		// Write should block on unbuffered channel
		select {
		case <-writeCompleted.Done():
			t.Fatal("Write should block on unbuffered channel")
		default:
			// Expected - write is blocked
		}

		// Read should unblock the write
		val := <-ch.Chan()
		assert.Equal(t, 42, val)

		<-writeCompleted.Done()
	})
}

func TestSafeChannel_Close_MultipleTimes(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	ctx, cancel := context.WithCancel(context.Background())
	ch := NewChannel[int](5, ctx)

	cancel()

	// Close multiple times should not panic
	ch.Close()
	ch.Close()
	ch.Close()
}

func TestSafeChannel_Close_ProperShutdownSequence(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	ctx, cancel := context.WithCancel(context.Background())
	ch := NewChannel[int](5, ctx)

	// Write some items
	require.NoError(t, ch.Write(1))
	require.NoError(t, ch.Write(2))

	// Proper shutdown sequence
	cancel()
	ch.Close()

	// Should still be able to read existing items
	assert.Equal(t, 1, <-ch.Chan())
	assert.Equal(t, 2, <-ch.Chan())

	// Channel should be closed now
	val, ok := <-ch.Chan()
	assert.False(t, ok, "channel should be closed")
	assert.Equal(t, 0, val)
}

func TestSafeChannel_ConcurrentWrites(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := NewChannel[int](100, ctx)

	numGoroutines := 10
	numWrites := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch multiple goroutines writing concurrently
	for i := range numGoroutines {
		go func(offset int) {
			defer wg.Done()
			for j := range numWrites {
				err := ch.Write(offset*numWrites + j)
				assert.NoError(t, err)
			}
		}(i)
	}

	// Read all items in another goroutine
	received := set.NewIntSet()
	var readerWg sync.WaitGroup
	readerWg.Add(1)
	go func() {
		defer readerWg.Done()
		for range numGoroutines * numWrites {
			val := <-ch.Chan()
			received.Add(val)
		}
	}()

	wg.Wait()
	readerWg.Wait()

	// Verify all items were received
	assert.Len(t, received, numGoroutines*numWrites)
}

func TestSafeChannel_ConcurrentWritesAndClose(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	// This test ensures there are no panics when writing and closing concurrently
	for range 100 {
		ctx, cancel := context.WithCancel(context.Background())
		ch := NewChannel[int](10, ctx)

		writeStarted := concurrency.NewSignal()

		var wg sync.WaitGroup
		wg.Add(2)

		// Writer goroutine
		go func() {
			defer wg.Done()
			for j := range 100 {
				if j == 0 {
					writeStarted.Signal()
				}
				_ = ch.Write(j)
			}
		}()

		// Closer goroutine - wait for writes to start, then close while writing
		go func() {
			defer wg.Done()
			<-writeStarted.Done()
			cancel()
			ch.Close()
		}()

		wg.Wait()
	}
}

func TestSafeChannel_WriteBlockedThenWaitableTriggered(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		ch := NewChannel[int](1, ctx)

		// Fill the channel
		require.NoError(t, ch.Write(1))

		// Start a write that will block
		writeStarted := concurrency.NewSignal()
		writeResult := make(chan error, 1)
		go func() {
			writeStarted.Signal()
			err := ch.Write(2)
			writeResult <- err
		}()

		// Wait for write to start
		<-writeStarted.Done()

		// Wait for the goroutine to become blocked
		synctest.Wait()

		// Trigger the waitable while write is blocked
		cancel()

		// The blocked write should return ErrWaitableTriggered
		err := <-writeResult
		assert.ErrorIs(t, err, ErrWaitableTriggered)
	})
}

func TestSafeChannel_WithStructTypes(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	type Event struct {
		ID   int
		Data string
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := NewChannel[Event](5, ctx)

	event1 := Event{ID: 1, Data: "test1"}
	event2 := Event{ID: 2, Data: "test2"}

	require.NoError(t, ch.Write(event1))
	require.NoError(t, ch.Write(event2))

	receivedEvent1 := <-ch.Chan()
	receivedEvent2 := <-ch.Chan()

	assert.Equal(t, event1, receivedEvent1)
	assert.Equal(t, event2, receivedEvent2)
}

func TestSafeChannel_WithPointerTypes(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	type Event struct {
		ID   int
		Data string
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := NewChannel[*Event](5, ctx)

	event1 := &Event{ID: 1, Data: "test1"}
	event2 := &Event{ID: 2, Data: "test2"}

	require.NoError(t, ch.Write(event1))
	require.NoError(t, ch.Write(event2))

	receivedEvent1 := <-ch.Chan()
	receivedEvent2 := <-ch.Chan()

	assert.Same(t, event1, receivedEvent1)
	assert.Same(t, event2, receivedEvent2)
}

func TestSafeChannel_NewSafeChannel_PanicsOnNilWaitable(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	// Creating a SafeChannel with a nil waitable should panic
	assert.Panics(t, func() {
		NewChannel[int](5, nil)
	}, "NewChannel should panic when waitable is nil")
}

func TestSafeChannel_Close_PanicsOnUntriggeredWaitable(t *testing.T) {
	goleak.AssertNoGoroutineLeaks(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := NewChannel[int](5, ctx)

	// Calling Close without triggering the waitable should panic
	assert.Panics(t, func() {
		ch.Close()
	}, "Close should panic when waitable has not been triggered")
}
