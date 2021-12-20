package errorhelpers

import (
	"github.com/pkg/errors"
)

// Sentinel errors for generic error classes. Must be convertible to gRPC's
// status.Status (via the respective interceptor) and hence also mapped to HTTP
// status codes. Note that error messages are the only reliable indicator of
// the error type in some cases, e.g., when embedded into a GraphQL response,
// thus changing them might break error matching for some clients, e.g., the UI.
//
// The list should be kept in sync with `ToGrpcCode()`.
var (
	// ErrAlreadyExists indicates that a object already exists.
	ErrAlreadyExists = errors.New("already exists")

	// ErrInvalidArgs indicates that a request has invalid arguments.
	ErrInvalidArgs = errors.New("invalid arguments")

	// ErrNotFound indicates that the requested object was not found.
	ErrNotFound = errors.New("not found")

	// ErrInvariantViolation indicates that some internal invariant has been
	// violated and the underlying component is in an inconsistent state.
	ErrInvariantViolation = errors.New("invariant violation")

	// ErrReferencedByAnotherObject indicates that the requested object cannot
	// be removed because it is referred to / in use by another object.
	ErrReferencedByAnotherObject = errors.New("referenced by another object")

	ErrNotAuthenticated = errors.New("not authenticated")

	// ErrNotAuthorized occurs if credentials are found, but they are
	// insufficiently authorized.
	ErrNotAuthorized = errors.New("not authorized")
)
