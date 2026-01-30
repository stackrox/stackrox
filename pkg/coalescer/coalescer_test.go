package coalescer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoalescer_ConcurrentCallsCoalesced(t *testing.T) {
	c := New[string]()
	var callCount atomic.Int32
	barrier := make(chan struct{})

	const numGoroutines = 10
	results := make([]string, numGoroutines)
	errs := make([]error, numGoroutines)

	var wg, startWg sync.WaitGroup
	startWg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			startWg.Done()
			startWg.Wait() // Wait for all goroutines to start
			results[idx], errs[idx] = c.Coalesce(context.Background(), "key", func() (string, error) {
				callCount.Add(1)
				<-barrier
				return "result", nil
			})
		}(i)
	}

	startWg.Wait()
	// Give goroutines time to call Coalesce
	time.Sleep(10 * time.Millisecond)
	close(barrier)
	wg.Wait()

	assert.Equal(t, int32(1), callCount.Load(), "function should be called exactly once")
	for i := 0; i < numGoroutines; i++ {
		assert.NoError(t, errs[i])
		assert.Equal(t, "result", results[i])
	}
}

func TestCoalescer_DifferentKeysNotCoalesced(t *testing.T) {
	c := New[string]()
	var callCount atomic.Int32

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		key := string(rune('a' + i))
		go func(key string) {
			defer wg.Done()
			_, err := c.Coalesce(context.Background(), key, func() (string, error) {
				callCount.Add(1)
				return key, nil
			})
			assert.NoError(t, err)
		}(key)
	}
	wg.Wait()

	assert.Equal(t, int32(3), callCount.Load(), "each key should trigger a separate call")
}

func TestCoalescer_ContextCancellation(t *testing.T) {
	c := New[string]()
	barrier := make(chan struct{})
	fnStarted := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-fnStarted
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := c.Coalesce(ctx, "key", func() (string, error) {
		close(fnStarted)
		<-barrier // Block forever
		return "result", nil
	})

	assert.ErrorIs(t, err, context.Canceled)
	close(barrier) // Cleanup
}

func TestCoalescer_ContextDeadlineExceeded(t *testing.T) {
	c := New[string]()
	barrier := make(chan struct{})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := c.Coalesce(ctx, "key", func() (string, error) {
		<-barrier // Block forever
		return "result", nil
	})

	assert.ErrorIs(t, err, context.DeadlineExceeded)
	close(barrier) // Cleanup
}

func TestCoalescer_ErrorPropagation(t *testing.T) {
	c := New[string]()
	expectedErr := errors.New("test error")

	const numGoroutines = 5
	errs := make([]error, numGoroutines)

	var wg, startWg sync.WaitGroup
	startWg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			startWg.Done()
			startWg.Wait()
			_, errs[idx] = c.Coalesce(context.Background(), "key", func() (string, error) {
				return "", expectedErr
			})
		}(i)
	}

	wg.Wait()

	for i := 0; i < numGoroutines; i++ {
		assert.ErrorIs(t, errs[i], expectedErr)
	}
}

func TestCoalescer_Forget(t *testing.T) {
	c := New[string]()
	var callCount atomic.Int32

	// First call
	result1, err := c.Coalesce(context.Background(), "key", func() (string, error) {
		callCount.Add(1)
		return "first", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "first", result1)

	// Forget the key
	c.Forget("key")

	// Second call should trigger a new execution
	result2, err := c.Coalesce(context.Background(), "key", func() (string, error) {
		callCount.Add(1)
		return "second", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "second", result2)

	assert.Equal(t, int32(2), callCount.Load(), "Forget should allow new execution")
}

func TestCoalescer_CancelledCallerDoesNotAffectOthers(t *testing.T) {
	c := New[string]()
	barrier := make(chan struct{})
	fnStarted := make(chan struct{})

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2 := context.Background()

	var result1, result2 string
	var err1, err2 error
	var wg sync.WaitGroup

	// First caller - will be cancelled
	wg.Add(1)
	go func() {
		defer wg.Done()
		result1, err1 = c.Coalesce(ctx1, "key", func() (string, error) {
			close(fnStarted)
			<-barrier
			return "result", nil
		})
	}()

	// Wait for fn to start
	<-fnStarted

	// Second caller - should succeed
	wg.Add(1)
	go func() {
		defer wg.Done()
		result2, err2 = c.Coalesce(ctx2, "key", func() (string, error) {
			t.Error("second caller should not execute fn")
			return "", nil
		})
	}()

	// Give second caller time to join the flight
	time.Sleep(10 * time.Millisecond)

	// Cancel first caller
	cancel1()

	// Give first caller time to return
	time.Sleep(10 * time.Millisecond)

	// Complete the function
	close(barrier)
	wg.Wait()

	// First caller should get context.Canceled
	assert.ErrorIs(t, err1, context.Canceled)
	assert.Empty(t, result1)

	// Second caller should get the result
	assert.NoError(t, err2)
	assert.Equal(t, "result", result2)
}

func TestCoalescer_TypedResults(t *testing.T) {
	type customType struct {
		value int
		name  string
	}

	c := New[*customType]()

	result, err := c.Coalesce(context.Background(), "key", func() (*customType, error) {
		return &customType{value: 42, name: "test"}, nil
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 42, result.value)
	assert.Equal(t, "test", result.name)
}

func TestCoalescer_NilResult(t *testing.T) {
	type customType struct{}

	c := New[*customType]()

	result, err := c.Coalesce(context.Background(), "key", func() (*customType, error) {
		return nil, nil
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}
