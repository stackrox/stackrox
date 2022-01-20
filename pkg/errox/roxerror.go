package errox

import (
	"fmt"
)

// RoxError is the common error interface.
type RoxError interface {
	error
	Base() RoxError
}

type baseError struct {
	message string
}

// new returns a new sentinel RoxError.
func new(message string) RoxError {
	return &baseError{message}
}

// Error returns error message. Implements error interface.
func (e *baseError) Error() string {
	return e.message
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

// New creates a new error based on base error.
func New(base RoxError, message string) RoxError {
	return &customError{base, nil, message}
}

// Error returns error message. Implements error interface.
func (e *customError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s", e.message, e.cause.Error())
	}
	return e.message
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
