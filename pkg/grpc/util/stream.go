package util

import (
	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StreamWithContext returns a grpc.ServerStream that has the given context.
func StreamWithContext(newCtx context.Context, stream grpc.ServerStream) grpc.ServerStream {
	if newCtx == stream.Context() {
		// Avoid creating a new ServerStream object based on the assumption that most callsites do not actually
		// modify the context.
		return stream
	}
	return &grpc_middleware.WrappedServerStream{
		ServerStream:   stream,
		WrappedContext: newCtx,
	}
}

// WithDefaultStatusCode applies the given default status code if err is a non-gRPC status error.
func WithDefaultStatusCode(err error, code codes.Code) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if ok {
		return st.Err()
	}
	return status.Error(code, err.Error())
}
