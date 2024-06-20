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

// WithSensitivef is a wrapper over WithSensitive.
func WithSensitivef(format string, args ...any) sensitiveErrorOption {
	return WithSensitive(fmt.Errorf(format, args...))
}
