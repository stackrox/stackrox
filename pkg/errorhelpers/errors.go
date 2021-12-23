package errorhelpers

import (
	"github.com/stackrox/rox/pkg/errox"
)

// TODO: replace the usage of these errors and functions with the ones from errox package.

var (
	// ErrAlreadyExists indicates that a object already exists.
	ErrAlreadyExists = errox.AlreadyExists

	// ErrInvalidArgs indicates that a request has invalid arguments.
	ErrInvalidArgs = errox.InvalidArgs

	// ErrNotFound indicates that the requested object was not found.
	ErrNotFound = errox.NotFound

	// ErrReferencedByAnotherObject indicates that the requested object cannot
	// be removed because it is referred to / in use by another object.
	ErrReferencedByAnotherObject = errox.ReferencedByAnotherObject

	// ErrInvariantViolation indicates that some internal invariant has been
	// violated and the underlying component is in an inconsistent state.
	ErrInvariantViolation = errox.InvariantViolation

	// ErrNoCredentials occurs if no credentials can be found.
	ErrNoCredentials = errox.NoCredentials

	// ErrNoValidRole occurs if no valid role can be found for user.
	ErrNoValidRole = errox.NoValidRole

	// ErrNotAuthorized occurs if credentials are found, but they are
	// insufficiently authorized.
	ErrNotAuthorized = errox.NotAuthorized

	// ErrNoAuthzConfigured occurs if authorization is not implemented for a
	// service. This is a programming error.
	ErrNoAuthzConfigured = errox.NoAuthzConfigured
)

// GenericNoValidRole wraps errox.NoValidRole with a generic error message.
func GenericNoValidRole() error {
	return errox.GenericNoValidRole()
}

// NewErrNotAuthorized wraps errox.NotAuthorized into an explanation.
func NewErrNotAuthorized(explanation string) error {
	return errox.NewErrNotAuthorized(explanation)
}

// NewErrNoCredentials wraps ErrNoCredentials into an explanation.
func NewErrNoCredentials(explanation string) error {
	return errox.NewErrNoCredentials(explanation)
}

// NewErrInvariantViolation wraps errox.InvariantViolation into an explanation.
func NewErrInvariantViolation(explanation string) error {
	return errox.NewErrInvariantViolation(explanation)
}

// NewErrInvalidArgs wraps errox.InvalidArgs into an explanation.
func NewErrInvalidArgs(explanation string) error {
	return errox.NewErrInvalidArgs(explanation)
}
