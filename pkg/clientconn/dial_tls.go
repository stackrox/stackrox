package clientconn

import (
	"context"
	"crypto/tls"

	"golang.stackrox.io/grpc-http1/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// DialTLSFunc is a function for establishing a gRPC connection.
type DialTLSFunc func(ctx context.Context, endpoint string, tlsClientConf *tls.Config, opts ...grpc.DialOption) (*grpc.ClientConn, error)

// DialTLS establishes a gRPC connection to the given endpoint, optionally using TLS for securing the transport layer
// and the given dial options.
// Note: if tlsClientConf is nil, the options *must* contain `WithInsecure()` or `WithTransportCredentials(insecure.NewCredentials())`,
// otherwise the connection will fail.
func DialTLS(ctx context.Context, endpoint string, tlsClientConf *tls.Config, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	allOpts := make([]grpc.DialOption, 0, len(opts)+1)
	if tlsClientConf != nil {
		allOpts = append(allOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsClientConf)))
	}
	allOpts = append(allOpts, opts...)

	return grpc.DialContext(ctx, endpoint, allOpts...)
}

// DialTLSWebSocket establishes a gRPC connection via a WebSocket proxy to the given endpoint,
// optionally using TLS for securing the transport layer and the given dial options.
// Note: if tlsClientConf is nil, the options *must* contain `WithInsecure()` or `WithTransportCredentials(insecure.NewCredentials())`,
// otherwise the connection will fail.
func DialTLSWebSocket(ctx context.Context, endpoint string, tlsClientConf *tls.Config, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return client.ConnectViaProxy(ctx, endpoint, tlsClientConf, client.DialOpts(opts...), client.UseWebSocket(true))
}
