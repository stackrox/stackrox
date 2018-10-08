package util

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
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
