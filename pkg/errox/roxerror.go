package errox

type roxError struct {
	message string
	base    error
}

// makeSentinel returns a new sentinel error. Semantically this is very close to
// `errors.New(message)` from the standard library.
func makeSentinel(message string) *roxError {
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
