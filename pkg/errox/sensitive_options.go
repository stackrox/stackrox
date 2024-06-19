package errox

import (
	"fmt"

	"github.com/pkg/errors"
)

type sensitiveErrorOptions struct {
	public    string
	sensitive error
}

type sensitiveErrorOption func(o *sensitiveErrorOptions)

// NewSensitive constructs a Sensitive error based on the provided options.
//
// Example:
//
//	err := errors.New("message")
//	err = NewSensitive(
//	  WithPublicError("public", err),
//	  WithSensitivef("secret %v", "1.2.3.4"))
//
//	err.Error() // "public: message"
//	UnconcealSensitive(err) // "secret 1.2.3.4: message"
func NewSensitive(opts ...sensitiveErrorOption) error {
	x := sensitiveErrorOptions{}
	for _, o := range opts {
		o(&x)
	}

	if x.sensitive == nil {
		if x.public == "" {
			return nil
		}
		return errors.New(x.public)
	}
	return MakeSensitive(x.public, x.sensitive)
}

// WithPublicMessage sets the public part of the resulting sensitive error.
func WithPublicMessage(message string) sensitiveErrorOption {
	return func(o *sensitiveErrorOptions) {
		o.public = message
	}
}

// WithPublicError adds a public message to the resulting sensitive error,
// including the public message from the provided error.
// The provided error will also be wrapped by the sensitive part.
//
// Example:
//
//	err := NewSensitive(
//	  WithPublicError("public", errors.New("error")),
//	  WithSensitive(errors.New("secret")))
//	err.Error() // "public: error"
//	UnconcealSensitive(err) // "secret: error"
func WithPublicError(public string, err error) sensitiveErrorOption {
	return func(o *sensitiveErrorOptions) {
		var serr SensitiveError
		WithSensitive(err)(o)
		if errors.As(err, &serr) {
			// The public message from the sensitive error will be added
			// automatically in Error().
			WithPublicMessage(public)(o)
		} else {
			WithPublicMessage(public + ": " + err.Error())(o)
		}
	}
}

// WithSensitive conceals the provided error and adds it to the sensitive part
// of the resulting sensitive error.
func WithSensitive(err error) sensitiveErrorOption {
	err = ConcealSensitive(err)
	return func(o *sensitiveErrorOptions) {
		if o.sensitive != nil {
			if o.public == "" {
				o.sensitive = errors.WithMessage(err, UnconcealSensitive(o.sensitive))
			} else {
				// If sensitive has already been provided by WithPublicError,
				// wrap it with the provided message.
				o.sensitive = errors.WithMessage(o.sensitive, UnconcealSensitive(err))
			}
		} else {
			o.sensitive = err
		}
	}
}

// WithSensitivef is a helping wrapper over WithSensitive.
func WithSensitivef(format string, args ...any) sensitiveErrorOption {
	return WithSensitive(fmt.Errorf(format, args...))
}
