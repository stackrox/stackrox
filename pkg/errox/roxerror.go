package errox

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

// Ensure roxError to implement RoxError.
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
