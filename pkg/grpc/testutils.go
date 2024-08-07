package grpc

import (
	"context"
	"net"
	"testing"

	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// CreateTestGRPCStreamingService creates a streaming server, registers the target
// services there, and returns a connection to the streaming server along with
// a function to close the connection.
func CreateTestGRPCStreamingService(
	ctx context.Context,
	_ testing.TB,
	registerServices func(registrar grpc.ServiceRegistrar),
) (*grpc.ClientConn, func(), error) {
	bufferSize := 1024 * 1024
	listener := bufconn.Listen(bufferSize)

	authInterceptor := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &grpcMiddleware.WrappedServerStream{
			ServerStream:   ss,
			WrappedContext: ctx,
		})
	}

	server := grpc.NewServer(grpc.StreamInterceptor(authInterceptor))
	registerServices(server)

	go func() {
		utils.IgnoreError(func() error { return server.Serve(listener) })
	}()

	conn, err := grpc.DialContext(ctx, "",
		grpc.WithContextDialer(
			func(ctx context.Context, _ string) (net.Conn, error) {
				return listener.DialContext(ctx)
			},
		),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	closeFunc := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}
	return conn, closeFunc, nil
}
