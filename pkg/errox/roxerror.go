// Package errox implements tooling and an interface for project errors
// handling, and a list of base sentinel errors.
//
// # Usage
//
// Base new errors on one of the existing sentinel errors:
//
//	ObjectNotFound := errox.NotFound.New("object not found")
//
// Classify encountered errors by making them a cause:
//
//	err := parse(args)
//	return errox.InvalidArgs.CausedBy(err)
//
// Check error class:
//
//	if errors.Is(err, errox.InvalidArgs) ...
//
// Format error messages:
//
//	return errox.NotFound.Newf("file %q not found", filename)
//
// Create error factories for generic errors:
//
//	ErrInvalidAlgorithmF := func(alg string) errox.Error {
//	    return errox.InvalidArgs.Newf("invalid algorithm %q used", alg)
//	}
//	...
//	return ErrInvalidAlgorithmF("256")
package errox

import "fmt"

// Error is the interface for rox errors.
type Error interface {
	error
	Unwrap() error
	New(message string) *RoxError
	Newf(format string, args ...interface{}) *RoxError
	CausedBy(cause interface{}) error
}

// RoxError is the basic type for rox errors. Not intended to be instantiated
// outside this package, it is exported because some of its exported methods
// return RoxError / *RoxError.
type RoxError struct {
	message string
	base    error
}

// Ensure RoxError implements errox.Error.
var _ Error = (*RoxError)(nil)

// makeSentinel returns a new sentinel error. Semantically this is very close to
// `errors.New(message)` from the standard library.
func makeSentinel(message string) *RoxError {
	return &RoxError{message, nil}
}

// Error returns error message. Implements error interface.
func (e *RoxError) Error() string {
	return e.message
}

// Unwrap returns the base of the error.
func (e *RoxError) Unwrap() error {
	return e.base
}

// New creates an error based on the existing RoxError, but with the
// personalized error message. Essentially, it allows for preserving the
// error's base error in the chain but hide its message.
//
// Example:
//
//	ErrRecordNotFound := errox.NotFound.New("record not found")
//	ErrRecordNotFound.Error() == "record not found" // true
//	errors.Is(ErrRecordNotFound, errox.NotFound)    // true
func (e *RoxError) New(message string) *RoxError {
	// Return *RoxError instead of errox.Error to enable `go vet` checks.
	return &RoxError{message, e}
}

// Newf creates an error based on the existing RoxError, but with the
// personalized formatted error message. Essentially, it allows for preserving
// the error's base error in the chain but hide its message.
//
// Example:
//
//	ErrRecordNotFound := errox.NotFound.Newf("record <%d> not found", recordIndex)
//	ErrRecordNotFound.Error() == "record <5> not found" // true
//	errors.Is(ErrRecordNotFound, errox.NotFound)        // true
func (e *RoxError) Newf(format string, args ...interface{}) *RoxError {
	// Return *RoxError instead of errox.Error to enable `go vet` checks.
	return e.New(fmt.Sprintf(format, args...))
}

// CausedBy adds a cause to the RoxError. The resulting message is a combination
// of the rox error and the cause following a colon.
//
// Note, that if cause is an error chain, the chain is collapsed to the message
// and the cause class is dropped, i.e.:
//
//	errors.Is(err.CausedBy(cause), cause) == false
//
// Example:
//
//	return errox.InvalidArgs.CausedBy(err)
//
// or
//
//	return errox.InvalidArgument.CausedBy("unknown parameter")
func (e *RoxError) CausedBy(cause interface{}) error {
	return fmt.Errorf("%w: %v", e, cause)
}

// CausedByf adds a cause to the RoxError. The resulting message is a
// combination of the rox error and the cause message, formatted based on the
// provided format specifier and arguments, following a colon.
//
// Example:
//
//	return errox.InvalidArgs.CausedByf("unknown parameter %v", p)
func (e *RoxError) CausedByf(format string, args ...interface{}) error {
	return e.CausedBy(fmt.Sprintf(format, args...))
}
