package clientconn

import (
	"context"
	"crypto/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// DialTLSFunc is a function for establishing a gRPC connection.
type DialTLSFunc func(ctx context.Context, endpoint string, tlsClientConf *tls.Config, opts ...grpc.DialOption) (*grpc.ClientConn, error)

// DialTLS establishes a gRPC connection to the given endpoint, using TLS for securing the transport layer
// and the given dial options.
func DialTLS(ctx context.Context, endpoint string, tlsClientConf *tls.Config, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	allOpts := make([]grpc.DialOption, 0, len(opts)+1)
	allOpts = append(allOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsClientConf)))
	allOpts = append(allOpts, opts...)

	return grpc.DialContext(ctx, endpoint, allOpts...)
}
