package errox

import (
	"fmt"

	"github.com/pkg/errors"
)

// Code is the generic error code type.
type Code int

// Generic Rox error codes.
const (
	CodeOK Code = iota
	CodeAlreadyExists
	CodeInvalidArgs
	CodeNotFound
	CodeReferencedByAnotherObject
	CodeInvariantViolation
	CodeNoCredentials
	CodeNoValidRole
	CodeNotAuthorized
	CodeNoAuthzConfigured
	CodeResourceAccessDenied

	// When adding a new code, consider updating the existing translations to other codes, like GRPC.
	// CodeUnknown must be the last one.
	CodeUnknown
)

// RoxError is the common error interface.
// Package errors are invited to implement this interface so the error translations
// could convert such errors to other types.
type RoxError interface {
	error
	Code() Code
	Base() RoxError
}

type errRox struct {
	code    Code
	base    RoxError
	cause   error
	message string
}

// New returns a new RoxError.
func New(code Code, message string) RoxError {
	return &errRox{code, nil, nil, message}
}

// NewCustom creates a new error based on base error.
func NewCustom(base RoxError, message string) RoxError {
	return &errRox{base.Code(), base, nil, message}
}

// Error return error message. Implements error interface.
func (e *errRox) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s", e.message, e.cause.Error())
	}
	return e.message
}

// Code returns error code. Implements RoxError interface.
func (e *errRox) Code() Code {
	return e.code
}

// Base returns the error base of the error. Implements RoxError interface.
func (e *errRox) Base() RoxError {
	return e.base
}

// Wrap wraps an error by a RoxError. Returns a new error.
func Wrap(cause error, rox RoxError) error {
	return &errRox{
		code:    rox.Code(),
		base:    rox,
		cause:   cause,
		message: rox.Error(),
	}
}

// Unwrap returns the cause of the error.
func (e *errRox) Unwrap() error {
	return e.cause
}

// Is returns true if e is or based on err.
func (e *errRox) Is(err error) bool {
	if re := RoxError(nil); errors.As(err, &re) {
		// Climb by the hierarchy to find the matching base.
		// Don't confuse with Unwrap().
		var base RoxError = e
		for ; base != nil; base = base.Base() {
			if re == base {
				return true
			}
		}
	}
	return false
}
