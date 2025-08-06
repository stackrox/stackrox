package eventual

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		v := New[string]()
		assert.False(t, v.IsSet())
		assert.Equal(t, "<not set>", v.String())

		go v.Set("value")
		assert.Equal(t, "value", v.Get(), "should wait for the value")
		assert.True(t, v.IsSet())
		assert.Equal(t, "value", v.String())
		assert.Equal(t, "value", v.Get())

		go v.Set("new value")
		assert.EventuallyWithT(t,
			func(collect *assert.CollectT) {
				assert.Equal(collect, "new value", v.Get())
			},
			5*time.Second, time.Millisecond,
		)
		assert.True(t, v.IsSet())
		assert.Equal(t, "new value", v.String())
	})

	t.Run("nil", func(t *testing.T) {
		var v *Value[string]
		assert.False(t, v.IsSet())
		assert.Equal(t, "", v.Get())
		assert.Equal(t, "<not set>", v.String())
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
}

func TestOptions(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		v := New[string](WithValue("value"))
		assert.True(t, v.IsSet())
		assert.Equal(t, "value", v.Get())
	})

	t.Run("with value and timeout", func(t *testing.T) {
		v := New[string](WithValue("value"), WithTimeout(time.Millisecond))

		assert.EventuallyWithT(t, func(c *assert.CollectT) {
			assert.True(c, v.IsSet())
			assert.Equal(c, "value", v.Get())
		}, time.Second, time.Millisecond)
	})

	t.Run("call OnTimeout", func(t *testing.T) {
		var timeout atomic.Bool
		v := New[string](
			WithValue("timeout"),
			WithTimeout(time.Millisecond),
			WithOnTimeout(func(set bool) {
				timeout.Store(set)
			}))

		assert.EventuallyWithT(t, func(c *assert.CollectT) {
			assert.True(c, v.IsSet())
			assert.Equal(c, "timeout", v.Get())
			assert.True(c, timeout.Load())
		}, time.Second, time.Millisecond)

		v.Set("value")
		assert.Equal(t, "value", v.Get())
	})

	t.Run("timeout without value", func(t *testing.T) {
		var timeout atomic.Bool
		v := New[string](
			WithTimeout(time.Millisecond),
			WithOnTimeout(func(set bool) {
				timeout.Store(set)
			}))

		assert.EventuallyWithT(t, func(c *assert.CollectT) {
			assert.True(c, v.IsSet())
			assert.Equal(c, "", v.Get())
			assert.True(c, timeout.Load())
		}, time.Second, time.Millisecond)
	})

	t.Run("set before timeout", func(t *testing.T) {
		var timeout atomic.Bool
		var called atomic.Bool
		v := New[string](
			WithValue("timeout"),
			WithTimeout(time.Second),
			WithOnTimeout(func(set bool) {
				called.Store(true)
				timeout.Store(set)
			}))
		assert.False(t, v.IsSet())
		v.Set("value")
		assert.False(t, called.Load())
		assert.True(t, v.IsSet())

		assert.EventuallyWithT(t, func(c *assert.CollectT) {
			assert.True(c, called.Load())
		}, 2*time.Second, time.Millisecond)
		assert.False(t, timeout.Load())
	})
}

func Test_panicInTest(t *testing.T) {
	if !buildinfo.ReleaseBuild {
		assert.Panics(t, func() {
			_ = New[string](WithValue(42))
		})
	} else {
		v := New[string](WithValue(42))
		assert.True(t, v.IsSet())
		assert.Equal(t, "", v.Get())
	}
}
