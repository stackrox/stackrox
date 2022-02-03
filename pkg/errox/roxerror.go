package errox

import (
	"fmt"
)

type roxError struct {
	message string
	base    error
}

// makeSentinel returns a new sentinel error.
func makeSentinel(message string) error {
	return &roxError{message, nil}
}

// Error returns error message. Implements error interface.
func (e *roxError) Error() string {
	return e.message
}

// Unwrap returns the base of the error.
func (e *roxError) Unwrap() error {
	return e.base
}

// New creates a new error based on base error.
func New(base error, message string) error {
	return &roxError{message, base}
}

// Newf creates a new error based on base error with formatted message.
func Newf(base error, format string, args ...interface{}) error {
	return New(base, fmt.Sprintf(format, args...))
}
