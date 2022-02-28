package errox

import (
	"fmt"
)

// New creates an error based on the existing roxError, but with the
// personalized error message. Essentially, it allows for preserving the error
// base error in the chain but hide its message.
//
// Example:
//     var ErrRecordNotFound := errox.NotFound.New("record not found")
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
