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
//   - The zero value of Value[T] is safe to use (Get() returns zero-value T).
type Value[T any] = *value[T]

//go:nocopy
type noCopy struct{}

// Lock and Unlock are no-ops used by go vet's copylocks check.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// box is a value container that allows for storing nil interface values to
// atomic.Value as non-nil to detect if the atomic.Value is unset.
type box[T any] struct {
	value T
}

// value[T] is the actual implementation of Value[T].
type value[T any] struct {
	_ noCopy

	// The current value.
	value atomic.Value // box[T]
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
func New[T any](opts ...Option[T]) Value[T] {
	var o options[T]
	for _, opt := range opts {
		opt(&o)
	}

	v := &value[T]{
		ready:        make(chan struct{}),
		defaultValue: o.defaultValue,
	}

	if o.defaultValue == nil {
		var zeroValue T
		v.defaultValue = &zeroValue
	}

	if o.context == nil && o.defaultValue != nil {
		// Ex.: New(WithDefaultValue(true)).
		v.Set(*o.defaultValue)
		return v
	}

	if o.context != nil {
		if _, ok := o.context.Deadline(); ok {
			go v.awaitContextDone(&o)
		}
	}

	// contextCancel is set only by WithTimeout().
	// Call it when the value is set before the timeout.
	if o.contextCancel != nil {
		go func() {
			<-v.ready
			o.contextCancel()
		}()
	}
	return v
}

func (v *value[T]) awaitContextDone(o *options[T]) {
	<-o.context.Done()
	swapped := v.value.CompareAndSwap(nil, box[T]{*v.defaultValue})
	if swapped {
		close(v.ready)
	}
	for _, f := range o.contextCallbacks {
		f(o.context, swapped)
	}
}

// IsSet returns true if the value has been set at least once.
// Returns false on nil Value pointer.
func (v *value[T]) IsSet() bool {
	return v != nil && v.value.Load() != nil
}

// Set initializes or overrides the current value.
// It unblocks all potentially waiting Get().
func (v *value[T]) Set(value T) {
	if v.value.Swap(box[T]{value}) == nil {
		close(v.ready)
	}
}

func (v *value[T]) zeroT() (zero T) { return }

// Get waits for the value to be set at least once, and returns the current
// value.
func (v *value[T]) Get() T {
	if v == nil {
		return v.zeroT()
	}
	<-v.ready
	return v.load()
}

func (v *value[T]) load() T {
	return v.value.Load().(box[T]).value
}

// Maybe returns immediately the set value and true, or default value and false.
func (v *value[T]) Maybe() (T, bool) {
	if v == nil {
		return v.zeroT(), false
	}
	if v.IsSet() {
		return v.load(), true
	}
	return *v.defaultValue, false
}

// GetWithContext is like Get(), but with context. If the context had been done
// before the value was set, the default value will be returned, and the state
// of the Value object will not be changed: IsSet() will return false.
func (v *value[T]) GetWithContext(ctx context.Context) T {
	if v == nil {
		return v.zeroT()
	}
	select {
	case <-v.ready:
		return v.load()
	case <-ctx.Done():
		return *v.defaultValue
	}
}

// Now returns an immediately initialized value.
func Now[T any](value T) Value[T] {
	return New(WithDefaultValue(value))
}
