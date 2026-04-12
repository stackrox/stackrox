package grpc

import (
	"context"
	"net"

	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
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

type debugLogger interface {
	Log(args ...any)
	Logf(format string, args ...any)
}

type debugLoggerImpl struct {
	log debugLogger
}

func (d *debugLoggerImpl) Log(args ...any) {
	if d == nil || d.log == nil {
		return
	}
	d.log.Log(args)
}

func (d *debugLoggerImpl) Logf(format string, args ...any) {
	if d == nil || d.log == nil {
		return
	}
	d.log.Logf(format, args)
}

// printSocketInfo is a debug hook that tests override with socket diagnostic output.
// Production code calls printSocketInfo(nil); tests reassign with a *testing.T-aware function.
var printSocketInfo = func(_ any) {}

var (
	procFiles = []string{"/proc/net/tcp", "/proc/net/tcp6"}
)
