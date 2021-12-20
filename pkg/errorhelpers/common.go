package errorhelpers

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	// ErrNoAuthzConfigured occurs if authorization is not implemented for a
	// service. This is a programming error.
	ErrNoAuthzConfigured = errors.New("service authorization is misconfigured")

	// ErrNoCredentials occurs if no credentials can be found.
	ErrNoCredentials = errors.New("credentials not found")

	// ErrNoValidRole occurs if no valid role can be found for user.
	ErrNoValidRole = errors.New("no valid role")
)

// GenericNoValidRole wraps ErrNoValidRole with a generic error message
func GenericNoValidRole() error {
	return fmt.Errorf("Access for this user is not authorized: %w. Please contact a system administrator.",
		ErrNoValidRole)
}

// NewErrNotAuthorized wraps ErrNotAuthorized into an explanation.
func NewErrNotAuthorized(explanation string) error {
	return fmt.Errorf("%w: %s", ErrNotAuthorized, explanation)
}

// NewErrNoCredentials wraps ErrNoCredentials into an explanation.
func NewErrNoCredentials(explanation string) error {
	return fmt.Errorf("%w: %s", ErrNoCredentials, explanation)
}

// NewErrInvariantViolation wraps ErrInvariantViolation into an explanation.
func NewErrInvariantViolation(explanation string) error {
	return fmt.Errorf("%w: %s", ErrInvariantViolation, explanation)
}

// NewErrInvalidArgs wraps ErrInvalidArgs into an explanation.
func NewErrInvalidArgs(explanation string) error {
	return fmt.Errorf("%w: %s", ErrInvalidArgs, explanation)
}
