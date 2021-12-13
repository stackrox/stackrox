package errorhelpers

import (
	"fmt"

	"google.golang.org/grpc/codes"
)

// Sentinel errors for generic error classes. Must be convertible to gRPC's
// status.Status (via the respective interceptor) and hence also mapped to HTTP
// status codes. Note that error messages are the only reliable indicator of
// the error type in some cases, e.g., when embedded into a GraphQL response,
// thus changing them might break error matching for some clients, e.g., the UI.
var (
	// ErrAlreadyExists indicates that the object already exists.
	ErrAlreadyExists = New(codes.AlreadyExists, "already exists")

	// ErrInvalidArgs indicates that a request has invalid arguments.
	ErrInvalidArgs = New(codes.InvalidArgument, "invalid arguments")

	// ErrNotFound indicates that the requested object was not found.
	ErrNotFound = New(codes.NotFound, "not found")

	// ErrReferencedByAnotherObject indicates that the requested object cannot
	// be removed because it is referred to / in use by another object.
	ErrReferencedByAnotherObject = New(codes.FailedPrecondition, "referenced by another object")

	// ErrInvariantViolation indicates that some internal invariant has been
	// violated and the underlying component is in an inconsistent state.
	ErrInvariantViolation = New(codes.Internal, "invariant violation")

	// ErrNoCredentials occurs if no credentials can be found.
	ErrNoCredentials = New(codes.Unauthenticated, "credentials not found")

	// ErrNoValidRole occurs if no valid role can be found for user.
	ErrNoValidRole = New(codes.Unauthenticated, "no valid role")

	// ErrNotAuthorized occurs if credentials are found, but they are
	// insufficiently authorized.
	ErrNotAuthorized = New(codes.PermissionDenied, "not authorized")

	// ErrNoAuthzConfigured occurs if authorization is not implemented for a
	// service. This is a programming error.
	ErrNoAuthzConfigured = New(codes.Unimplemented, "service authorization is misconfigured")
)

// GenericNoValidRole wraps ErrNoValidRole with a generic error message.
func GenericNoValidRole() error {
	return fmt.Errorf("access for this user is not authorized: %w, please contact your system administrator",
		ErrNoValidRole)
}

// NewErrNotAuthorized wraps ErrNotAuthorized into an explanation.
func NewErrNotAuthorized(explanation string) error {
	return ErrNotAuthorized.Wrapf(explanation)
}

// NewErrInvariantViolation wraps ErrInvariantViolation into an explanation.
func NewErrInvariantViolation(explanation string) error {
	return ErrInvariantViolation.Wrapf(explanation)
}

// NewErrInvalidArgs wraps ErrInvalidArgs into an explanation.
func NewErrInvalidArgs(explanation string) error {
	return ErrInvalidArgs.Wrapf(explanation)
}
