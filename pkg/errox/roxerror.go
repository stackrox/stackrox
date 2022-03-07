// Package errox implements tooling and an interface for project errors
// handling, and a list of base sentinel errors.
//
// Usage
//
// Base new errors on one of the existing sentinel errors:
//     ObjectNotFound := errox.NotFound.New("object not found")
//
// Classify encountered errors by making them a cause:
//     err := parse(args)
//     return errox.InvalidArgs.CausedBy(err)
//
// Check error class:
//     if errors.Is(err, errox.InvalidArgs) ...
//
// Format error messages:
//     return errox.NotFound.New(fmt.Sprintf("file '%s' not found", filename))
//
// Create error factories for generic errors:
//     ErrInvalidAlgorithmF := func(alg string) RoxError {
//         return errox.InvalidArgs.New(
//             fmt.Sprintf("invalid algorithm %q used", alg))
//     }
//     ...
//     return ErrInvalidAlgorithmF("256")
package errox

import "fmt"

// RoxError is the interface of rox errors.
type RoxError interface {
	error
	Unwrap() error
	New(message string) RoxError
	CausedBy(cause interface{}) error
}

type roxError struct {
	message string
	base    error
}

// Ensure roxError implements RoxError.
var _ RoxError = (*roxError)(nil)

// makeSentinel returns a new sentinel error. Semantically this is very close to
// `errors.New(message)` from the standard library.
func makeSentinel(message string) RoxError {
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

// New creates an error based on the existing roxError, but with the
// personalized error message. Essentially, it allows for preserving the error
// base error in the chain but hide its message.
//
// Example:
//     ErrRecordNotFound := errox.NotFound.New("record not found")
//     ErrRecordNotFound.Error() == "record not found" // true
//     errors.Is(ErrRecordNotFound, errox.NotFound)    // true
func (e *roxError) New(message string) RoxError {
	return &roxError{message, e}
}

// CausedBy adds a cause to the roxError. The resulting message is a combination
// of the rox error and the cause following a colon.
//
// Example:
//     return errox.InvalidArgument.CausedBy(err)
// or
//     return errox.InvalidArgument.CausedBy("unknown parameter")
func (e *roxError) CausedBy(cause interface{}) error {
	return fmt.Errorf("%w: %v", e, cause)
}
