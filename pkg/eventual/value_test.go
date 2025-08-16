package eventual

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		v := New[string]()
		assert.False(t, v.IsSet())

		go v.Set("value")
		assert.Equal(t, "value", v.Get(), "should wait for the value")
		assert.True(t, v.IsSet())
		assert.Equal(t, "value", v.Get())

		go v.Set("new value")
		assert.EventuallyWithT(t,
			func(collect *assert.CollectT) {
				assert.Equal(collect, "new value", v.Get())
			},
			5*time.Second, time.Millisecond,
		)
		assert.True(t, v.IsSet())
	})

	t.Run("nil", func(t *testing.T) {
		var v Value[string]
		assert.False(t, v.IsSet())
		assert.Equal(t, "", v.Get())
	})

	t.Run("multiple readers", func(t *testing.T) {
		v := New[string]()

		const n = 10
		resultCh := make(chan string)
		for range n {
			go func() {
				resultCh <- v.Get()
			}()
		}
		v.Set("value")
		for range n {
			assert.Equal(t, "value", <-resultCh)
		}
	})

	t.Run("pointer type", func(t *testing.T) {
		var i *int
		v := New(WithDefaultValue(i))
		assert.True(t, v.IsSet())
		assert.NotPanics(t, func() { v.Set(nil) })
		assert.Nil(t, v.Get())

		v = New[*int]()
		assert.False(t, v.IsSet())
		assert.NotPanics(t, func() { v.Set(nil) })
		assert.Nil(t, v.Get())
	})

	t.Run("any", func(t *testing.T) {
		v := New[any]()
		assert.False(t, v.IsSet())
		assert.NotPanics(t, func() { v.Set(nil) })
		// This would hang without boxing.
		assert.Nil(t, v.Get())
	})
}

func TestValue_Maybe(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var v Value[int]
		m, ok := v.Maybe()
		assert.False(t, ok)
		assert.Empty(t, m)
	})

	t.Run("not nil", func(t *testing.T) {
		v := New(WithDefaultValue("default").WithContext(context.Background()))

		m, ok := v.Maybe()
		assert.Equal(t, "default", m)
		assert.False(t, ok)
		assert.False(t, v.IsSet())

		v.Set("value")
		m, ok = v.Maybe()
		assert.Equal(t, "value", m)
		assert.True(t, ok)
		assert.True(t, v.IsSet())
	})
}

func TestValue_GetWithContext(t *testing.T) {
	t.Run("not set", func(t *testing.T) {
		v := New(WithDefaultValue("default").WithContext(context.Background()))
		ctx, cancel := context.WithCancelCause(context.Background())
		var value string
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			value = v.GetWithContext(ctx)
			wg.Done()
		}()
		cancel(errors.New("a cause"))
		wg.Wait()

		assert.False(t, v.IsSet())
		v.Set("value")

		assert.Equal(t, "default", value)
		assert.Equal(t, "value", v.Get())

		if assert.Error(t, ctx.Err()) {
			assert.Equal(t, "context canceled", ctx.Err().Error())
			assert.Equal(t, "a cause", context.Cause(ctx).Error())
		}
	})
	t.Run("set", func(t *testing.T) {
		v := New(WithDefaultValue("default").WithContext(context.Background()))
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		var value string
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			value = v.GetWithContext(ctx)
			wg.Done()
		}()
		v.Set("value")
		wg.Wait()
		assert.Equal(t, "value", value)
	})
	t.Run("nil", func(t *testing.T) {
		var v Value[string]
		ctx, cancel := context.WithCancel(context.Background())
		assert.False(t, v.IsSet())
		assert.Equal(t, "", v.GetWithContext(ctx))
		cancel()
		assert.Equal(t, "", v.GetWithContext(ctx))
	})
}

func TestNow(t *testing.T) {
	v := Now("value")
	assert.True(t, v.IsSet())
	assert.Equal(t, "value", v.Get())
	assert.Equal(t, "value", v.GetWithContext(context.Background()))

	vb := Now(true)
	assert.True(t, vb.IsSet())
	assert.True(t, vb.Get())
	assert.True(t, vb.GetWithContext(context.Background()))
}
