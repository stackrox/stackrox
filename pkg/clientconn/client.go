package clientconn

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// GRPCConnection returns a grpc.ClientConn object.
func GRPCConnection(endpoint string) (conn *grpc.ClientConn, err error) {
	return AuthenticatedGRPCConnection(endpoint)
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
		ServerName:   mtls.CentralCN.String(), // This is required!
	}
	creds := credentials.NewTLS(tlsConfig)
	return grpc.Dial(endpoint, grpc.WithTransportCredentials(creds), keepAliveDialOption())
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

// Parameters for keep alive.
func keepAliveDialOption() grpc.DialOption {
	// Since we are holding open a GRPC stream, enable keep alive.
	// Ping every minute of inactivity, and wait 30 seconds. Do this even when no streams are open (though
	// one should always be open with central.)
	params := keepalive.ClientParameters{
		Time:                1 * time.Minute,
		Timeout:             30 * time.Second,
		PermitWithoutStream: true,
	}
	return grpc.WithKeepaliveParams(params)
}
