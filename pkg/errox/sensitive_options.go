package errox

import (
	"fmt"

	"github.com/pkg/errors"
)

type sensitiveErrorOption func(o *RoxSensitiveError)

// NewSensitive constructs a Sensitive error based on the provided options.
//
// Example:
//
//	err := NewSensitive(
//	  WithSensitive(dnsError),
//	  WithPublicMessage("message"),
//	)
//
//	err.Error() // "message: lookup: DNS error"
//	UnconcealSensitive(err) // "lookup localhost on 127.0.0.1: DNS error"
func NewSensitive(opts ...sensitiveErrorOption) error {
	result := RoxSensitiveError{}
	for _, o := range opts {
		o(&result)
	}
	if result.sensitive == nil {
		if result.public == nil {
			return nil
		}
		return result.public
	}
	result.unprotectedGoRoutineID.Store(unsetID)
	return &result
}

// WithPublicMessage adds the public part to the resulting sensitive error.
// See WithPublicError.
func WithPublicMessage(message string) sensitiveErrorOption {
	return WithPublicError(errors.New(message))
}

// WithPublicError adds the public part to the resulting sensitive error,
// wrapping already set value.
//
// Example:
//
//	err := NewSensitive(
//	  WithPublicError(errors.New("one")),
//	  WithPublicError(errors.New("two")))
//	err.Error() // "two: one"
func WithPublicError(err error) sensitiveErrorOption {
	return func(o *RoxSensitiveError) {
		if o.public != nil {
			o.public = errors.WithMessage(o.public, err.Error())
		} else {
			o.public = err
		}
	}
}

// WithSensitive adds the provided error to the sensitive part of the resulting
// sensitive error.
func WithSensitive(err error) sensitiveErrorOption {
	return func(o *RoxSensitiveError) {
		if o.sensitive != nil {
			o.sensitive = errors.WithMessage(o.sensitive, UnconcealSensitive(err))
		} else {
			o.sensitive = ConcealSensitive(err)
		}
	}
}

// WithSensitivef adds a formatted error message to the sensitive part.
// If public part has been set already, it will be wrapped to the sensitive part
// as well.
//
// Example:
//
//	err := NewSensitive(WithPublicMessage("message"), WithSensitivef("secret"))
//	err.Error() // "message"
//	UnconcealSensitive(err) // "secret: message"
func WithSensitivef(format string, args ...any) sensitiveErrorOption {
	return func(o *RoxSensitiveError) {
		if o.public != nil {
			WithSensitive(o.public)(o)
		}
		WithSensitive(fmt.Errorf(format, args...))(o)
	}
}
