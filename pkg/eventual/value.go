package eventual

import (
	"context"

	"sync/atomic"
)

// Value[T] is a thread-safe container for a value that may be provided later.
// Get() blocks until Set() is called (or, if configured, until the provided
// Context is done, in which case the default value from WithDefaultValue() is
// used).
//
// Key points:
//   - Use New to create a Value[T] (with optional default, timeout, etc.).
//   - Calling Set() unblocks all pending Get() calls and updates the stored
//     value.
//   - Get() always returns the latest value; subsequent Set() calls overwrite
//     it.
//   - The zero value of *Value[T] is safe to use (Get() returns zero-value T).
//   - Value[T] must not be copied after first use.
type Value[T any] struct {
	// The current value.
	value atomic.Value
	// The channel is closed when the value is set.
	ready chan struct{}
	// The value to return if context is done, but the value is not set.
	defaultValue *T
}

// New constructs an eventually initialized value of type T.
// Example:
//
//	v := New[string]()
//	go v.Set("value")
//	fmt.Println(v.Get()) // output: value
func New[T any](opts ...Option[T]) *Value[T] {
	var o options[T]
	for _, opt := range opts {
		opt(&o)
	}
	v := &Value[T]{
		ready: make(chan struct{}),
	}

	if o.defaultValue != nil {
		v.defaultValue = o.defaultValue
	} else {
		var zeroValue T
		v.defaultValue = &zeroValue
	}

	if o.context == nil && o.defaultValue != nil {
		// Ex.: New(WithDefaultValue(true))
		v.Set(*o.defaultValue)
		return v
	}

	if o.context != nil {
		go func() {
			<-o.context.Done()
			swapped := v.value.CompareAndSwap(nil, *v.defaultValue)
			if swapped {
				v.close()
			}
			for _, f := range o.contextCallbacks {
				f(o.context, swapped)
			}
		}()
	}

	// contextCancel is set only by WithTimeout().
	// Call it when the value is set before the timeout.
	if o.contextCancel != nil {
		go func() {
			v.Get()
			o.contextCancel()
		}()
	}
	return v
}

// IsSet returns true if the value has been set at least once.
// Returns false on nil Value pointer.
func (v *Value[T]) IsSet() bool {
	return v != nil && v.value.Load() != nil
}

// Set initializes or overrides the current value.
// It unblocks all potentially waiting Get().
func (v *Value[T]) Set(value T) {
	v.value.Store(value)
	v.close()
}

func (v *Value[T]) close() {
	select {
	case <-v.ready:
		// already closed.
	default:
		close(v.ready)
	}
}

// Get waits for the value to be set at least once, and returns the current
// value.
func (v *Value[T]) Get() T {
	if v == nil {
		var zeroValue T
		return zeroValue
	}
	<-v.ready
	return v.value.Load().(T)
}

// Maybe returns immediately the set value and true, or default value and false.
func (v *Value[T]) Maybe() (T, bool) {
	if v == nil {
		var zeroValue T
		return zeroValue, false
	}
	if v.IsSet() {
		return v.Get(), true
	}
	return *v.defaultValue, false
}

// GetWithContext is like Get(), but with context. If the context had been done
// before the value was set, the default value will be returned, and the state
// of the Value object will not be changed: IsSet() will return false.
func (v *Value[T]) GetWithContext(ctx context.Context) T {
	if v == nil {
		var zeroValue T
		return zeroValue
	}
	select {
	case <-v.ready:
		return v.value.Load().(T)
	case <-ctx.Done():
		return *v.defaultValue
	}
}

// Now returns an immediately initialized value.
func Now[T any](value T) *Value[T] {
	return New(WithDefaultValue(value))
}
