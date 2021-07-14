package errors

import (
	"context"
	"errors"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorToGrpcCodeInterceptor translates common errors defined in errorhelpers to GRPC codes.
func ErrorToGrpcCodeInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	return resp, errToGrpcError(err)
}

// ErrorToGrpcCodeStreamInterceptor translates common errors defined in errorhelpers to GRPC codes.
func ErrorToGrpcCodeStreamInterceptor(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := handler(srv, ss)
	return errToGrpcError(err)
}

func errToGrpcError(err error) error {
	_, ok := status.FromError(err)
	// ok is true for nil and status
	if ok {
		return err
	}
	code := errorTypeToGrpcCode(err)
	return status.New(code, err.Error()).Err()
}

func errorTypeToGrpcCode(err error) codes.Code {
	switch {
	case errors.Is(err, errorhelpers.ErrNotFound):
		return codes.NotFound
	case errors.Is(err, errorhelpers.ErrInvalidArgs):
		return codes.InvalidArgument
	case errors.Is(err, errorhelpers.ErrAlreadyExists):
		return codes.AlreadyExists
	case errors.Is(err, errorhelpers.ErrReferencedByAnotherObject):
		return codes.FailedPrecondition
	case errors.Is(err, errorhelpers.ErrInvariantViolation):
		return codes.Internal
	case errors.Is(err, sac.ErrResourceAccessDenied):
		return codes.PermissionDenied
	default:
		return codes.Internal
	}
}
