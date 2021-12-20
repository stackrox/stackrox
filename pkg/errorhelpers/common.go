package errorhelpers

import (
	"fmt"

	"github.com/pkg/errors"
)

// NOTE: These errors can (and should?) be moved to appropriate packages. If an
// error becomes common across multiple packages and/or components, it should be
// moved to the _true_ sentinels list, currently in "custom_types.go".
var (
	// ErrNoAuthzConfigured occurs if authorization is not implemented for a
	// service. This is a programming error.
	ErrNoAuthzConfigured = errors.New("service authorization is misconfigured")

	// ErrNoCredentials occurs if no credentials can be found.
	ErrNoCredentials = errors.New("credentials not found")

	// ErrNoValidRole occurs if no valid role can be found for user.
	ErrNoValidRole = errors.New("no valid role")
)

func Explain(sentinel error, explanation string) error {
	return fmt.Errorf("%w: %s", sentinel, explanation)
}

// GenericNoValidRole wraps ErrNoValidRole with a generic error message
func GenericNoValidRole() error {
	return fmt.Errorf("Access for this user is not authorized: %w. Please contact a system administrator.",
		ErrNoValidRole)
}

////////////////////////////////////////////////////////////////////////////////
// Consider removing the functions below in favour of direct use of           //
// `Explain()`.                                                               //
//

// NewErrNotAuthorized wraps ErrNotAuthorized into an explanation.
func NewErrNotAuthorized(explanation string) error {
	return Explain(ErrNotAuthorized, explanation)
}

// NewErrNoCredentials wraps ErrNoCredentials into an explanation.
func NewErrNoCredentials(explanation string) error {
	return Explain(ErrNoCredentials, explanation)
}

// NewErrInvariantViolation wraps ErrInvariantViolation into an explanation.
func NewErrInvariantViolation(explanation string) error {
	return Explain(ErrInvariantViolation, explanation)
}

// NewErrInvalidArgs wraps ErrInvalidArgs into an explanation.
func NewErrInvalidArgs(explanation string) error {
	return Explain(ErrInvalidArgs, explanation)
}
