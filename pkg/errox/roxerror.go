package errox

import (
	"fmt"
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
type RoxError interface {
	error
	Code() Code
	Base() RoxError
}

type baseError struct {
	code    Code
	message string
}

// New returns a new RoxError.
func New(code Code, message string) RoxError {
	return &baseError{code, message}
}

// Error return error message. Implements error interface.
func (e *baseError) Error() string {
	return e.message
}

// Code returns error code. Implements RoxError interface.
func (e *baseError) Code() Code {
	return e.code
}

// Base returns the error base of the error. Implements RoxError interface.
func (e *baseError) Base() RoxError {
	return nil
}

type customError struct {
	base    RoxError
	cause   error
	message string
}

// NewCustom creates a new error based on base error.
func NewCustom(base RoxError, message string) RoxError {
	return &customError{base, nil, message}
}

// Error return error message. Implements error interface.
func (e *customError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s", e.message, e.cause.Error())
	}
	return e.message
}

// Code returns error code. Implements RoxError interface.
func (e *customError) Code() Code {
	return e.base.Code()
}

// Base returns the error base of the error. Implements RoxError interface.
func (e *customError) Base() RoxError {
	return e.base
}

// Wrap wraps an error by a RoxError. Returns a new error. Panics if rox is nil.
func Wrap(cause error, rox RoxError) RoxError {
	return &customError{
		base:    rox,
		cause:   cause,
		message: rox.Error(),
	}
}

// Unwrap returns the cause of the error.
func (e *customError) Unwrap() error {
	return e.cause
}

// Is returns true if e is or based on err.
func (e *customError) Is(err error) bool {
	// Climb by the hierarchy to find the matching base.
	// Don't confuse with Unwrap().
	for base := RoxError(e); base != nil; base = base.Base() {
		if err == base {
			return true
		}
	}
	return false
}
