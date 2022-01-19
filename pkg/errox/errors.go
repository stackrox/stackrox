package errox

import (
	"fmt"
)

// Sentinel errors for generic error classes.
// Note that error messages are the only reliable indicator of
// the error type in some cases, e.g., when embedded into a GraphQL response,
// thus changing them might break error matching for some clients, e.g., the UI.
var (
	AlreadyExists             = new("already exists")
	InvalidArgs               = new("invalid arguments")
	NotFound                  = new("not found")
	ReferencedByAnotherObject = new("referenced by another object")
	InvariantViolation        = new("invariant violation")
	NoCredentials             = new("credentials not found")
	NoValidRole               = new("no valid role")
	NotAuthorized             = new("not authorized")
	NoAuthzConfigured         = new("service authorization is misconfigured")
	// When adding a new error please update the translators (gRPC, HTTP etc.).
)

// GenericNoValidRole wraps ErrNoValidRole with a generic error message.
func GenericNoValidRole() error {
	return fmt.Errorf("access for this user is not authorized: %w, please contact your system administrator",
		NoValidRole)
}

func explain(err error, explanation string) error {
	return fmt.Errorf("%w: %s", err, explanation)
}

// NewErrNotAuthorized wraps ErrNotAuthorized into an explanation.
func NewErrNotAuthorized(explanation string) error {
	return explain(NotAuthorized, explanation)
}

// NewErrNoCredentials wraps ErrNoCredentials into an explanation.
func NewErrNoCredentials(explanation string) error {
	return explain(NoCredentials, explanation)
}

// NewErrInvariantViolation wraps ErrInvariantViolation into an explanation.
func NewErrInvariantViolation(explanation string) error {
	return explain(InvariantViolation, explanation)
}

// NewErrInvalidArgs wraps ErrInvalidArgs into an explanation.
func NewErrInvalidArgs(explanation string) error {
	return explain(InvalidArgs, explanation)
}
