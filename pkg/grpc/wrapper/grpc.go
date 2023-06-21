package wrapper

import (
	"context"
	"net"

	"google.golang.org/grpc"
)

// GRPCWrapper provides a wrapper for a server interface similar to http.Server.
type GRPCWrapper struct {
	internal *grpc.Server
}

// Serve wraps gRPC Serve function.
func (w *GRPCWrapper) Serve(l net.Listener) error {
	return w.internal.Serve(l)
}

// Shutdown maps to gRPC GracefulStop routine.
func (w *GRPCWrapper) Shutdown(_ context.Context) error {
	w.internal.GracefulStop()
	return nil
}

func GRPC(server *grpc.Server) *GRPCWrapper {
	return &GRPCWrapper{internal: server}
}
