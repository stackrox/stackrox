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

func WithPublicMessage(message string) sensitiveErrorOption {
	return func(o *sensitiveErrorOptions) {
		o.public = message
	}
}

// WithPublicError adds a public message to the error, including the public
// message from the given error.
func WithPublicError(public string, err error) sensitiveErrorOption {
	return func(o *sensitiveErrorOptions) {
		var serr SensitiveError
		if errors.As(err, &serr) {
			WithSensitive(err)(o)
			// The public message from the sensitive error will be added
			// automatically in Error().
			WithPublicMessage(public)(o)
		} else {
			WithPublicMessage(public + ": " + err.Error())(o)
		}
	}
}

func WithConcealed(err error) sensitiveErrorOption {
	return WithSensitive(ConcealSensitive(err))
}

func WithSensitive(err error) sensitiveErrorOption {
	return func(o *sensitiveErrorOptions) {
		if o.sensitive != nil {
			o.sensitive = errors.WithMessage(o.sensitive, UnconcealSensitive(err))
		} else {
			o.sensitive = err
		}
	}
}

func WithSensitivef(format string, args ...any) sensitiveErrorOption {
	return WithSensitive(fmt.Errorf(format, args...))
}

// NewSensitive constructs a Sensitive error based on the provided options.
//
// Example:
//
//	err := errors.New("message")
//	err = NewSensitive(
//	  WithPublicError("public", err),
//	  WithSensitivef(err, "secret %v", ip))
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
