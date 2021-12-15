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
	var re errox.RoxError
	if errors.As(err, &re) {
		switch re.Code() {
		case errox.CodeOK:
			return codes.OK
		case errox.CodeAlreadyExists:
			return codes.AlreadyExists
		case errox.CodeInvalidArgs:
			return codes.InvalidArgument
		case errox.CodeNotFound:
			return codes.NotFound
		case errox.CodeReferencedByAnotherObject:
			return codes.FailedPrecondition
		case errox.CodeInvariantViolation:
			return codes.Internal
		case errox.CodeNoCredentials:
			return codes.Unauthenticated
		case errox.CodeNoValidRole:
			return codes.Unauthenticated
		case errox.CodeNotAuthorized:
			return codes.PermissionDenied
		case errox.CodeNoAuthzConfigured:
			return codes.Unimplemented
		case errox.CodeResourceAccessDenied:
			return codes.PermissionDenied
		}
	}
	return codes.Internal
}
