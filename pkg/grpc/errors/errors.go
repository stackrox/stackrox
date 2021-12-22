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

// All errox codes except CodeUnknown are mapped to GRPC code.
var erroxToGRPCCode = map[errox.Code]codes.Code{
	errox.CodeOK:                        codes.OK,
	errox.CodeAlreadyExists:             codes.AlreadyExists,
	errox.CodeInvalidArgs:               codes.InvalidArgument,
	errox.CodeNotFound:                  codes.NotFound,
	errox.CodeReferencedByAnotherObject: codes.FailedPrecondition,
	errox.CodeInvariantViolation:        codes.Internal,
	errox.CodeNoCredentials:             codes.Unauthenticated,
	errox.CodeNoValidRole:               codes.Unauthenticated,
	errox.CodeNotAuthorized:             codes.PermissionDenied,
	errox.CodeNoAuthzConfigured:         codes.Unimplemented,
	errox.CodeResourceAccessDenied:      codes.PermissionDenied,
	errox.CodeUnknown:                   codes.Internal,
}

func grpcCode(err error) codes.Code {
	var re errox.RoxError
	if errors.As(err, &re) {
		if i, ok := erroxToGRPCCode[re.Code()]; ok {
			return i
		}
	}
	return codes.Internal
}
