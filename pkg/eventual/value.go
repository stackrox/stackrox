package eventual

import (
	"context"
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

	defaultValue *T
}

type options struct {
	defaultValue     any
	context          context.Context
	contextCancel    func()
	contextCallbacks []func(context.Context, bool)
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

	defaultValue, ok := o.defaultValue.(T)
	if o.defaultValue != nil && !ok {
		// Panic in debug...
		utils.Should(errors.New("wrong default eventual value type"))
		// ... but use the default value for the type T in release.
	}
	if ok {
		v.defaultValue = &defaultValue
	} else {
		var value T
		v.defaultValue = &value
	}
	if o.context == nil && o.defaultValue != nil {
		v.Set(defaultValue)
		return v
	}

	// contextCancel is set only by WithTimeout().
	// Call it when the value is set before the timeout.
	if o.contextCancel != nil {
		go func() {
			v.Get()
			o.contextCancel()
		}()
	}
	if o.context != nil {
		go func() {
			<-o.context.Done()
			swapped := v.value.CompareAndSwap(nil, defaultValue)
			if swapped {
				v.close()
			}
			for _, f := range o.contextCallbacks {
				f(o.context, swapped)
			}
		}()
	}
	return v
}

// WithDefaultValue provides the value to be set either on initialization, or
// after context cancellation.
func WithDefaultValue(value any) Option {
	return func(o *options) {
		o.defaultValue = value
	}
}

// WithContext provides the context. When the context is cancelled, the eventual
// value is set to the value, provided with `WithValue()`.
// If context callback is also provided, it will be called.
func WithContext(ctx context.Context) Option {
	return func(o *options) {
		o.context = ctx
	}
}

// WithContextCallback provides a function to be called after the timeout. The
// boolean argument will tell whether the value has been set on timeout.
// Multiple callbacks will be called in the order of the provided options.
func WithContextCallback(f func(_ context.Context, set bool)) Option {
	return func(o *options) {
		o.contextCallbacks = append(o.contextCallbacks, f)
	}
}

// WithTimeout provides the timeout after which the eventual value will be set
// to the value, provided with WithValue(), or to the default value for the
// type.
func WithTimeout(d time.Duration) Option {
	return func(o *options) {
		if o.context == nil {
			o.context = context.Background()
		}
		o.context, o.contextCancel = context.WithTimeout(o.context, d)
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

// GetWithContext is like Get(), but with context. If the context is cancelled
// before the value is set, the default value is returned.
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
	return New[T](WithDefaultValue(value))
}
