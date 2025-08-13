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

// Option configures Value[T] initialization.
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

// WithDefaultValue provides a value to be:
//   - set on initialization if no context or timeout provided;
//   - set after context is done if used with WithContext;
//   - set timeout expiration if WithTimeout() is provided;
//   - returned by GetWithContext() if the provided context is done.
func WithDefaultValue[T any](value T) Option[T] {
	return func(o *options[T]) {
		o.defaultValue = &value
	}
}

// WithContext provides the context. When the provided context is done:
//   - the Value is set to the default value;
//   - the context callback is called if provided with WithContextCallback().
func WithContext[T any](ctx context.Context) Option[T] {
	return func(o *options[T]) {
		o.context = ctx
	}
}

// WithContextCallback registers a callback to run when the context is done.
// The bool parameter indicates if the Value was set due to the context
// cancellation.
// Callbacks are invoked in the order they were added.
func WithContextCallback[T any](f func(_ context.Context, setDefault bool)) Option[T] {
	return func(o *options[T]) {
		o.contextCallbacks = append(o.contextCallbacks, f)
	}
}

// WithTimeout provides a timeout after which the Value will be set to the
// value, provided with WithDefaultValue(), or to the T zero value.
func WithTimeout[T any](d time.Duration) Option[T] {
	return func(o *options[T]) {
		if o.context == nil {
			o.context = context.Background()
		}
		o.context, o.contextCancel = context.WithTimeout(o.context, d)
	}
}
