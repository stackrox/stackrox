package eventual

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

// Value allows for an eventual value initialization.
// The value retrieval will be blocked until it is first initialized by calling
// Value.Set(), or after a timeout provided with WithTimeout().
// Consequent calls to Value.Set() update the value.
// The implementation is safe for concurrent access.
// Use New() to construct instances of this type.
// Must not be copied.
type Value[T any] struct {
	value atomic.Value
	ready chan struct{}
}

type options struct {
	timeout   time.Duration
	value     any
	onTimeout func(bool)
}

// Option configures the constructor behavior.
type Option func(*options)

// New constructs an eventually initialized value of type T.
// Example:
//
//	v := New[string]()
//	go v.Set("value")
//	fmt.Println(v.Get()) // output: value
func New[T any](opts ...Option) *Value[T] {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	v := &Value[T]{
		ready: make(chan struct{}),
	}

	defaultValue, ok := o.value.(T)
	if o.value != nil && !ok {
		// Panic in debug...
		utils.Should(errors.New("wrong default eventual value type"))
		// ... but use the default value for the type T in release.
	}
	if o.timeout == 0 {
		if o.value != nil {
			v.Set(defaultValue)
		}
		return v
	}
	time.AfterFunc(o.timeout, func() {
		swapped := v.value.CompareAndSwap(nil, defaultValue)
		if swapped {
			v.close()
		}
		if o.onTimeout != nil {
			o.onTimeout(swapped)
		}
	})
	return v
}

// WithValue provides the value to be set either on initialization, or after a
// timeout, if WithTimeout() is also provided.
// The value must be of the same type, as T of the Eventual[T].
func WithValue(value any) Option {
	return func(o *options) {
		o.value = value
	}
}

// WithTimeout provides the timeout after which the eventual value will be set
// to the value, provided with WithValue(), or to the default value for the
// type.
func WithTimeout(d time.Duration) Option {
	return func(o *options) {
		o.timeout = d
	}
}

// WithOnTimeout provides a function to be called after the timeout. The boolean
// argument will tell whether the value has been set on timeout.
func WithOnTimeout(f func(set bool)) Option {
	return func(o *options) {
		o.onTimeout = f
	}
}

// String representation of the internal value or "<not set>" if not set.
func (v *Value[T]) String() string {
	if v.IsSet() {
		return fmt.Sprintf("%v", v.Get())
	}
	return "<not set>"
}

// IsSet returns true if the value has been set at least once.
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

// Now returns an immediately initialized value.
func Now[T any](value T) *Value[T] {
	return New[T](WithValue(value))
}
