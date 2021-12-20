package errorhelpers

import (
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
)

func ToGrpcCode(err error) codes.Code {
	switch {
	case errors.Is(err, ErrAlreadyExists):
		return codes.AlreadyExists
	case errors.Is(err, ErrInvalidArgs):
		return codes.InvalidArgument
	case errors.Is(err, ErrNotFound):
		return codes.NotFound
	case errors.Is(err, ErrInvariantViolation):
		return codes.Internal
	case errors.Is(err, ErrReferencedByAnotherObject):
		return codes.FailedPrecondition
	case errors.Is(err, ErrNotAuthenticated):
		return codes.Unauthenticated
	case errors.Is(err, ErrNotAuthorized):
		return codes.PermissionDenied
	default:
		return codes.Internal
	}
}
