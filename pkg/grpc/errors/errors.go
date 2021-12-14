package errors

import (
	"errors"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/pkg/errorhelpers"
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
	var re errorhelpers.RoxError
	if errors.As(err, &re) {
		switch re.Code() {
		case errorhelpers.CodeOK:
			return codes.OK
		case errorhelpers.CodeAlreadyExists:
			return codes.AlreadyExists
		case errorhelpers.CodeInvalidArgs:
			return codes.InvalidArgument
		case errorhelpers.CodeNotFound:
			return codes.NotFound
		case errorhelpers.CodeReferencedByAnotherObject:
			return codes.FailedPrecondition
		case errorhelpers.CodeInvariantViolation:
			return codes.Internal
		case errorhelpers.CodeNoCredentials:
			return codes.Unauthenticated
		case errorhelpers.CodeNoValidRole:
			return codes.Unauthenticated
		case errorhelpers.CodeNotAuthorized:
			return codes.PermissionDenied
		case errorhelpers.CodeNoAuthzConfigured:
			return codes.Unimplemented
		case errorhelpers.CodeResourceAccessDenied:
			return codes.PermissionDenied
		}
	}
	return codes.Internal
}
