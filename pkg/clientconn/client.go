package clientconn

import (
	"crypto/tls"
	"fmt"

	"bitbucket.org/stack-rox/apollo/pkg/features"
	"bitbucket.org/stack-rox/apollo/pkg/mtls"
	"bitbucket.org/stack-rox/apollo/pkg/mtls/verifier"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GRPCConnection returns a grpc.ClientConn object.
func GRPCConnection(endpoint string) (conn *grpc.ClientConn, err error) {
	if features.MTLS.Enabled() {
		return AuthenticatedGRPCConnection(endpoint)
	}
	return UnauthenticatedGRPCConnection(endpoint)
}

// AuthenticatedGRPCConnection returns a grpc.ClientConn object that uses
// client certificates found on the local file system.
func AuthenticatedGRPCConnection(endpoint string) (conn *grpc.ClientConn, err error) {
	cert, err := mtls.LeafCertificateFromFile()
	if err != nil {
		return nil, fmt.Errorf("client credentials: %s", err)
	}
	pool, err := verifier.TrustedCertPool()
	if err != nil {
		return nil, fmt.Errorf("trusted pool: %s", err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		ServerName:   mtls.CentralName, // This is required!
	}
	creds := credentials.NewTLS(tlsConfig)
	return grpc.Dial(endpoint, grpc.WithTransportCredentials(creds))
}

// UnauthenticatedGRPCConnection returns a grpc.ClientConn object that does not use credentials.
// Deprecated: This is only to be used temporarily until Sensors
// issue certificates to their workers.
func UnauthenticatedGRPCConnection(endpoint string) (conn *grpc.ClientConn, err error) {
	tlsConfig := &tls.Config{
		// TODO(cg): Issue credentials and remove this.
		InsecureSkipVerify: true,
	}
	creds := credentials.NewTLS(tlsConfig)
	return grpc.Dial(endpoint, grpc.WithTransportCredentials(creds))
}
