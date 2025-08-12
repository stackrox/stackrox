package eventual

import (
	"context"
	"time"
)

type options[T any] struct {
	defaultValue     *T
	context          context.Context
	contextCancel    func()
	contextCallbacks []func(context.Context, bool)
}

// Option configures the constructor behavior.
type Option[T any] func(*options[T])

func WithType[T any]() Option[T] { return func(o *options[T]) {} }

func (o Option[T]) chain(next Option[T]) Option[T] {
	return func(opts *options[T]) {
		o(opts)
		next(opts)
	}
}

func (o Option[T]) WithContext(ctx context.Context) Option[T] {
	return o.chain(WithContext[T](ctx))
}

func (o Option[T]) WithContextCallback(f func(_ context.Context, set bool)) Option[T] {
	return o.chain(WithContextCallback[T](f))
}

func (o Option[T]) WithTimeout(d time.Duration) Option[T] {
	return o.chain(WithTimeout[T](d))
}

// WithDefaultValue provides the value to be set either on initialization, or
// after context cancellation.
func WithDefaultValue[T any](value T) Option[T] {
	return func(o *options[T]) {
		o.defaultValue = &value
	}
}

// WithContext provides the context. When the context is cancelled, the eventual
// value is set to the value, provided with `WithValue()`.
// If context callback is also provided, it will be called.
func WithContext[T any](ctx context.Context) Option[T] {
	return func(o *options[T]) {
		o.context = ctx
	}
}

// WithContextCallback provides a function to be called after the timeout. The
// boolean argument will tell whether the value has been set on timeout.
// Multiple callbacks will be called in the order of the provided options.
func WithContextCallback[T any](f func(_ context.Context, set bool)) Option[T] {
	return func(o *options[T]) {
		o.contextCallbacks = append(o.contextCallbacks, f)
	}
}

// WithTimeout provides the timeout after which the eventual value will be set
// to the value, provided with WithValue(), or to the default value for the
// type.
func WithTimeout[T any](d time.Duration) Option[T] {
	return func(o *options[T]) {
		if o.context == nil {
			o.context = context.Background()
		}
		o.context, o.contextCancel = context.WithTimeout(o.context, d)
	}
}
