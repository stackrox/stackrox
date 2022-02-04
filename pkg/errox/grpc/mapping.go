package grpc

import (
	"errors"

	"github.com/stackrox/rox/pkg/errox"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// unwrapGRPCStatus unwraps the `err` chain to find an error
// implementing `GRPCStatus()`.
func unwrapGRPCStatus(err error) *status.Status {
	var se interface{ GRPCStatus() *status.Status }
	if errors.As(err, &se) {
		return se.GRPCStatus()
	}
	return nil
}

// ErrToGRPCStatus wraps an error into a gRPC status with code.
func ErrToGRPCStatus(err error) *status.Status {
	if se, ok := status.FromError(err); ok {
		return se
	}
	var code codes.Code
	// `status.FromError()` doesn't unwrap the `err` chain, so unwrap it here.
	if se := unwrapGRPCStatus(err); se != nil {
		code = se.Code()
	} else {
		code = RoxErrorToGRPCCode(err)
	}
	return status.New(code, err.Error())
}

// RoxErrorToGRPCCode translates known sentinel errors to according gRPC codes.
func RoxErrorToGRPCCode(err error) codes.Code {
	switch {
	case err == nil:
		return codes.OK
	case errors.Is(err, errox.AlreadyExists):
		return codes.AlreadyExists
	case errors.Is(err, errox.InvalidArgs):
		return codes.InvalidArgument
	case errors.Is(err, errox.NotFound):
		return codes.NotFound
	case errors.Is(err, errox.ReferencedByAnotherObject):
		return codes.FailedPrecondition
	case errors.Is(err, errox.InvariantViolation):
		return codes.Internal
	case errors.Is(err, errox.NoCredentials):
		return codes.Unauthenticated
	case errors.Is(err, errox.NoValidRole):
		return codes.Unauthenticated
	case errors.Is(err, errox.NotAuthorized):
		return codes.PermissionDenied
	case errors.Is(err, errox.NoAuthzConfigured):
		return codes.Unimplemented
	default:
		return codes.Internal
	}
}
