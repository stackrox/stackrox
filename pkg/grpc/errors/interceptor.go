package errors

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorToGrpcCodeInterceptor translates common errors defined in errorhelpers to GRPC codes.
func ErrorToGrpcCodeInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	return resp, ErrToGrpcStatus(err).Err()
}

// ErrorToGrpcCodeStreamInterceptor translates common errors defined in errorhelpers to GRPC codes.
func ErrorToGrpcCodeStreamInterceptor(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := handler(srv, ss)
	return ErrToGrpcStatus(err).Err()
}

// ErrToHTTPStatus maps known internal and gRPC errors to the appropriate
// HTTP status code.
func ErrToHTTPStatus(err error) int {
	return runtime.HTTPStatusFromCode(ErrToGrpcStatus(err).Code())
}

// ErrToGrpcStatus wraps an error into a gRPC status with code.
func ErrToGrpcStatus(err error) *status.Status {
	if s, ok := status.FromError(err); ok {
		// `err` is either nil or status.Status.
		return s
	}
	code := codes.Internal
	if re, ok := errorhelpers.IsRoxError(err); ok {
		code = re.GRPCCode()
	}
	return status.New(code, err.Error())
}
