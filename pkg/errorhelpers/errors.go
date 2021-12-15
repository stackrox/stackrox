package errorhelpers

import (
	"fmt"

	"github.com/stackrox/rox/pkg/errox"
)

// TODO: replace the usage of these errors and functions with the ones from errox package.

var (
	// ErrInvalidArgs indicates that a request has invalid arguments.
	ErrInvalidArgs = errox.New(errox.CodeInvalidArgs, "", "invalid arguments")

	// ErrNotFound indicates that the requested object was not found.
	ErrNotFound = errox.New(errox.CodeNotFound, "", "not found")

	// ErrNoCredentials occurs if no credentials can be found.
	ErrNoCredentials = errox.New(errox.CodeNoCredentials, "", "credentials not found")
)

// GenericNoValidRole wraps errox.NoValidRole with a generic error message.
func GenericNoValidRole() error {
	return fmt.Errorf("access for this user is not authorized: %w, please contact your system administrator",
		errox.NoValidRole)
}

// NewErrNotAuthorized wraps errox.NotAuthorized into an explanation.
func NewErrNotAuthorized(explanation string) error {
	return fmt.Errorf("%w: %s", errox.NotAuthorized, explanation)
}

// NewErrNoCredentials wraps ErrNoCredentials into an explanation.
func NewErrNoCredentials(explanation string) error {
	return fmt.Errorf("%w: %s", errox.NoCredentials, explanation)
}

// NewErrInvariantViolation wraps errox.InvariantViolation into an explanation.
func NewErrInvariantViolation(explanation string) error {
	return fmt.Errorf("%w: %s", errox.InvariantViolation, explanation)
}

// NewErrInvalidArgs wraps errox.InvalidArgs into an explanation.
func NewErrInvalidArgs(explanation string) error {
	return fmt.Errorf("%w: %s", errox.InvalidArgs, explanation)
}
