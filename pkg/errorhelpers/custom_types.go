package errorhelpers

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	// ErrAlreadyExists indicates that a object already exists.
	ErrAlreadyExists = errors.New("already exists")

	// ErrInvalidArgs indicates that a request has invalid arguments.
	ErrInvalidArgs = errors.New("invalid arguments")

	// ErrNotFound indicates that the requested object was not found.
	ErrNotFound = errors.New("not found")

	// ErrReferencedByAnotherObject indicates that the requested object cannot
	// be removed because it is referred to / in use by another object.
	ErrReferencedByAnotherObject = errors.New("referenced by another object")

	// ErrInvariantViolation indicates that some internal invariant has been
	// violated and the underlying component is in an inconsistent state.
	ErrInvariantViolation = errors.New("invariant violation")

	// ErrNoCredentials occurs if no credentials can be found.
	ErrNoCredentials = errors.New("credentials not found")

	// ErrNotAuthorized occurs if credentials are found, but they are
	// insufficiently authorized.
	ErrNotAuthorized = errors.New("not authorized")

	// ErrNoAuthzConfigured occurs if authorization is not implemented for a
	// service. This is a programming error.
	ErrNoAuthzConfigured = errors.New("service authorization is misconfigured")
)

// NewErrNotAuthorized wraps ErrNotAuthorized into an explanation.
func NewErrNotAuthorized(explanation string) error {
	return fmt.Errorf("%w: %s", ErrNotAuthorized, explanation)
}
