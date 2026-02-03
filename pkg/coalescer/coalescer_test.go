package coalescer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoalescer_ContextCancellation(t *testing.T) {
	c := New[string]()
	barrier := make(chan struct{})
	fnStarted := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-fnStarted
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
