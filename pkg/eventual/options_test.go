package eventual

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

func TestWithType(t *testing.T) {
	var n int
	f1 := func(context.Context) {
		n = 1
	}
	f2 := func(context.Context) {
		n = 2
	}
	var opts options[string]
	type nothing struct{}

	WithType[string]().
		WithContext(context.WithValue(context.Background(), nothing{}, "value")).
		WithContextCallback(f1).
		WithContextCallback(f2).
		WithTimeout(time.Minute)(&opts)

	assert.Equal(t, "value", opts.context.Value(nothing{}))

	assert.NotNil(t, opts.context)
	opts.contextCancel()
	<-opts.context.Done()

	assert.NotNil(t, opts.contextCancel)

	if assert.Len(t, opts.contextCallbacks, 2) {
		opts.contextCallbacks[0](nil)
		assert.Equal(t, 1, n)
		opts.contextCallbacks[1](nil)
		assert.Equal(t, 2, n)
	}
}

func TestOptions(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		v := New(WithDefaultValue("value"))
		assert.True(t, v.IsSet())
		assert.Equal(t, "value", v.Get())
	})

	t.Run("with value and timeout", func(t *testing.T) {
		v := New(WithDefaultValue("value"), WithTimeout[string](time.Microsecond))

		assert.EventuallyWithT(t, func(c *assert.CollectT) {
			assert.True(c, v.IsSet())
			assert.Equal(c, "value", v.Get())
		}, time.Second, time.Millisecond)
	})

	t.Run("timeout without value", func(t *testing.T) {
		v := New(WithType[string]().WithTimeout(time.Millisecond))

		assert.EventuallyWithT(t, func(c *assert.CollectT) {
			assert.True(c, v.IsSet())
			assert.Equal(c, "", v.Get())
		}, time.Second, time.Millisecond)
	})

	t.Run("WithContext cause", func(t *testing.T) {
		t.Run("cancelled", func(t *testing.T) {
			var cause error
			var wg sync.WaitGroup
			wg.Add(1)
			ctx, cancel := context.WithCancelCause(context.Background())
			v := New(WithDefaultValue("value").
				WithContext(ctx).
				WithContextCallback(func(ctx context.Context) {
					cause = context.Cause(ctx)
					wg.Done()
				}))
			testCause := errors.New("test cause")
			cancel(testCause)
			wg.Wait()
			assert.Equal(t, "value", v.Get())
			assert.ErrorIs(t, cause, testCause)
		})
		t.Run("set", func(t *testing.T) {
			var called atomic.Bool
			ctx, cancel := context.WithCancel(context.Background())
			v := New(WithType[string]().
				WithContext(ctx),
				WithContextCallback[string](func(context.Context) {
					// Should not be called.
					called.Store(true)
				}))
			// Set doesn't affect the context.
			v.Set("value")
			cancel()
			assert.Equal(t, "value", v.Get())
			assert.False(t, called.Load())
		})
	})

	t.Run("WithTimeout cause", func(t *testing.T) {
		t.Run("timeout", func(t *testing.T) {
			var cause error
			var wg sync.WaitGroup
			wg.Add(1)
			v := New(WithDefaultValue("default").
				WithTimeout(time.Microsecond).
				WithContextCallback(func(ctx context.Context) {
					cause = context.Cause(ctx)
					wg.Done()
				}))
			wg.Wait()
			assert.Equal(t, "default", v.Get())
			assert.ErrorIs(t, cause, Timeout)
		})
		t.Run("set", func(t *testing.T) {
			var called atomic.Bool
			v := New(WithType[string]().
				WithTimeout(time.Hour).
				WithContextCallback(func(context.Context) {
					// Should not be called.
					called.Store(true)
				}))
			// Set cancels the timeout without a cause.
			v.Set("value")
			assert.Equal(t, "value", v.Get())
			assert.False(t, called.Load())
		})
	})

	t.Run("context and timeout", func(t *testing.T) {

		ctx, cancel := context.WithCancelCause(context.Background())

		var cause error
		var wg sync.WaitGroup
		wg.Add(1)
		v := New(WithDefaultValue("value").
			WithContext(ctx).
			WithTimeout(time.Hour).
			WithContextCallback(func(ctx context.Context) {
				cause = context.Cause(ctx)
				wg.Done()
			}))

		testCause := errors.New("test cause")
		cancel(testCause)
		wg.Wait()
		assert.Equal(t, "value", v.Get())
		assert.ErrorIs(t, cause, testCause)
	})
}
