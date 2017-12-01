package clientconn

import (
	"crypto/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GRPCConnection returns a grpc.ClientConn object.
func GRPCConnection(endpoint string) (conn *grpc.ClientConn, err error) {
	tlsConfig := &tls.Config{
		// TODO(cgorman) Proper cert management and TLS
		InsecureSkipVerify: true,
	}
	creds := credentials.NewTLS(tlsConfig)
	return grpc.Dial(endpoint, grpc.WithTransportCredentials(creds))
}
