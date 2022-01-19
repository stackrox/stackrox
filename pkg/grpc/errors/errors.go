package errors

import (
	"errors"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/pkg/errox"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func unwrapGRPCStatus(err error) *status.Status {
	if err == nil {
		return nil
	}
	if se, ok := err.(interface {
		GRPCStatus() *status.Status
	}); ok {
		return se.GRPCStatus()
	}
	return unwrapGRPCStatus(errors.Unwrap(err))
}

// ErrToGrpcStatus wraps an error into a gRPC status with code.
func ErrToGrpcStatus(err error) *status.Status {
	if err == nil {
		return nil
	}
	se, ok := status.FromError(err)
	if ok && se != nil {
		return se
	}
	code := grpcCode(err)
	if se = unwrapGRPCStatus(err); se != nil {
		code = se.Code()
	}
	return status.New(code, err.Error())
}

// ErrToHTTPStatus maps known internal and gRPC errors to the appropriate
// HTTP status code.
func ErrToHTTPStatus(err error) int {
	return runtime.HTTPStatusFromCode(ErrToGrpcStatus(err).Code())
}

func grpcCode(err error) codes.Code {
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
