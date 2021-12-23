package errox

import (
	"fmt"
)

// Sentinel errors for generic error classes. Must be convertible to gRPC's
// status.Status (via the respective interceptor) and hence also mapped to HTTP
// status codes. Note that error messages are the only reliable indicator of
// the error type in some cases, e.g., when embedded into a GraphQL response,
// thus changing them might break error matching for some clients, e.g., the UI.
var (
	// AlreadyExists indicates that the object already exists.
	AlreadyExists = New(CodeAlreadyExists, "already exists")

	// InvalidArgs indicates that a request has invalid arguments.
	InvalidArgs = New(CodeInvalidArgs, "invalid arguments")

	// NotFound indicates that the requested object was not found.
	NotFound = New(CodeNotFound, "not found")

	// ReferencedByAnotherObject indicates that the requested object cannot
	// be removed because it is referred to / in use by another object.
	ReferencedByAnotherObject = New(CodeReferencedByAnotherObject, "referenced by another object")

	// InvariantViolation indicates that some internal invariant has been
	// violated and the underlying component is in an inconsistent state.
	InvariantViolation = New(CodeInvariantViolation, "invariant violation")

	// NoCredentials occurs if no credentials can be found.
	NoCredentials = New(CodeNoCredentials, "credentials not found")

	// NoValidRole occurs if no valid role can be found for user.
	NoValidRole = New(CodeNoValidRole, "no valid role")

	// NotAuthorized occurs if credentials are found, but they are
	// insufficiently authorized.
	NotAuthorized = New(CodeNotAuthorized, "not authorized")

	// NoAuthzConfigured occurs if authorization is not implemented for a
	// service. This is a programming error.
	NoAuthzConfigured = New(CodeNoAuthzConfigured, "service authorization is misconfigured")
)

// GenericNoValidRole wraps ErrNoValidRole with a generic error message.
func GenericNoValidRole() error {
	return fmt.Errorf("access for this user is not authorized: %w, please contact your system administrator",
		NoValidRole)
}

// NewErrNotAuthorized wraps ErrNotAuthorized into an explanation.
func NewErrNotAuthorized(explanation string) error {
	return fmt.Errorf("%w: %s", NotAuthorized, explanation)
}

// NewErrInvariantViolation wraps ErrInvariantViolation into an explanation.
func NewErrInvariantViolation(explanation string) error {
	return fmt.Errorf("%w: %s", InvariantViolation, explanation)
}

// NewErrInvalidArgs wraps ErrInvalidArgs into an explanation.
func NewErrInvalidArgs(explanation string) error {
	return fmt.Errorf("%w: %s", InvalidArgs, explanation)
}
