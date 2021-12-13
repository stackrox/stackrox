package errorhelpers

import (
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RoxError is an error interface.
type RoxError interface {
	error
	Code() codes.Code
	Wrap(err error) RoxError
	Wraps(message string) RoxError
	Wrapf(explanation string) RoxError
}

// Implements RoxError interface.
type errRox struct {
	code     codes.Code
	message  string // may appear only if the original error is nil
	original error
}

// Error returns the error message. Implements error interface.
func (e *errRox) Error() string {
	if e.original != nil {
		return e.original.Error()
	}
	return e.message
}

// GRPCCode returns the GRPC code, associtated with the error.
func (e *errRox) Code() codes.Code {
	return e.code
}

// Unwrap returns the original error. Used by errors.Unwrap().
func (e *errRox) Unwrap() error {
	return e.original
}

// GRPCStatus can be called by grpc/status.FromError.
// Doesn't store the parent errors in the chain if the error is wrapped.
func (e *errRox) GRPCStatus() *status.Status {
	if e.original != nil {
		return status.New(e.code, e.original.Error())
	}
	return status.New(e.code, e.message)
}

// New returns a RoxError with the provided GRPC code and message.
func New(code codes.Code, message string) RoxError {
	return &errRox{
		code,
		message,
		nil,
	}
}

// Wrap adds the RoxError on top of the original error chain.
func (e *errRox) Wrap(original error) RoxError {
	return &errRox{
		e.code,
		e.message,
		original,
	}
}

// Wraps constructs a new error with the provided message and wraps it with the RoxError.
func (e *errRox) Wraps(message string) RoxError {
	return e.Wrap(errors.New(message))
}

// Wrapf constructs a new error with a combination of the error static message and the provided explanation,
// and wraps it with the RoxError.
func (e *errRox) Wrapf(explanation string) RoxError {
	return e.Wrap(fmt.Errorf("%s: %s", e.message, explanation))
}

// Is called by errors.Is(err, target).
// Returns true if the target error chain has a RoxError with the same code.
func (e *errRox) Is(target error) bool {
	var re RoxError
	return errors.As(target, &re) && e.code == re.Code()
}
