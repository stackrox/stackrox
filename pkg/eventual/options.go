package eventual

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// Timeout is the context cause on timeout, set with WithTimeout().
var Timeout = errors.New("set to default on timeout")

type options[T any] struct {
	defaultValue     *T
	context          context.Context
	contextCancel    func()
	contextCallbacks []func(context.Context)
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

func (o Option[T]) WithContextCallback(f func(_ context.Context)) Option[T] {
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
//
// Warning: this option must not be added after WithTimeout(), because the
// provided context overrides any previously set.
func WithContext[T any](ctx context.Context) Option[T] {
	return func(o *options[T]) {
		o.context = ctx
	}
}

// WithContextCallback registers a callback to run when the context is
// cancelled.
// The context Cause indicates the cause of the callback. If the value has been
// created with WithTimeout(), and the timeout expires, the cause is set to
// Timeout.
// If the Value is set before the context is cancelled, the callback is not
// called.
// Callbacks are invoked in the order they were added.
func WithContextCallback[T any](f func(_ context.Context)) Option[T] {
	return func(o *options[T]) {
		o.contextCallbacks = append(o.contextCallbacks, f)
	}
}

// WithTimeout provides a timeout after which the Value will be set to the
// value, provided with WithDefaultValue(), or to the T zero value.
//
// Warning: this option may conflict with WithContext() and must not be added
// more than once.
func WithTimeout[T any](d time.Duration) Option[T] {
	return func(o *options[T]) {
		if o.context == nil {
			o.context = context.Background()
		}
		o.context, o.contextCancel = context.WithTimeoutCause(o.context, d, Timeout)
	}
}
