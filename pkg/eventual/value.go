package eventual

import (
	"context"
	"fmt"
	"sync/atomic"
)

// Value allows for an eventual value initialization.
// The value retrieval will be blocked until it is first initialized by calling
// Value.Set(), or after context cancellation.
// Consequent calls to Value.Set() update the value.
// The implementation is safe for concurrent access.
// Use New() to construct instances of this type.
// Must not be copied.
type Value[T any] struct {
	// The current value.
	value atomic.Value
	// The channel is closed when the value is set.
	ready chan struct{}
	// The value to return on context cancellation.
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

	if o.context == nil && o.defaultValue != nil {
		v.Set(*o.defaultValue)
		return v
	}

	if o.defaultValue != nil {
		v.defaultValue = o.defaultValue
	} else {
		var value T
		v.defaultValue = &value
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

// String representation of the internal value or "<not set>" if not set.
func (v *Value[T]) String() string {
	if v.IsSet() {
		return fmt.Sprintf("%v", v.Get())
	}
	return "<not set>"
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
		var value T
		return value
	}
	<-v.ready
	return v.value.Load().(T)
}

// GetWithContext is like Get(), but with context. If the context had been
// cancelled before the value was set, the default value will be returned, and
// the state of the Value object will not be changed: IsSet() will return false.
func (v *Value[T]) GetWithContext(ctx context.Context) T {
	if v == nil {
		var value T
		return value
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
