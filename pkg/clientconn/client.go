package clientconn

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/grpc/authn/tokenbased"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// Service identifies the service which acts as gRPC server.
type Service int

const (
	// Central is the service name for central
	Central Service = iota
	// Sensor is the service name for sensor
	Sensor
)

func tlsConfig(clientCert tls.Certificate, rootCAs *x509.CertPool, server string) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		ServerName:   server,
		RootCAs:      rootCAs,
	}
}

// AuthenticatedGRPCConnection returns a grpc.ClientConn object that uses
// client certificates found on the local file system.
func AuthenticatedGRPCConnection(endpoint string, service Service) (conn *grpc.ClientConn, err error) {
	clientCert, err := mtls.LeafCertificateFromFile()
	if err != nil {
		return nil, fmt.Errorf("client credentials: %s", err)
	}
	rootCAs, err := verifier.TrustedCertPool()
	if err != nil {
		return nil, fmt.Errorf("trusted pool: %s", err)
	}

	var creds credentials.TransportCredentials
	switch service {
	case Central:
		creds = credentials.NewTLS(tlsConfig(clientCert, rootCAs, mtls.CentralSubject.Hostname()))
	case Sensor:
		creds = credentials.NewTLS(tlsConfig(clientCert, rootCAs, mtls.SensorSubject.Hostname()))
	}
	return grpc.Dial(endpoint, grpc.WithTransportCredentials(creds), keepAliveDialOption())
}

// GRPCConnectionWithBasicAuth returns a grpc.ClientConn using the given username/password to authenticate
// via basic auth.
func GRPCConnectionWithBasicAuth(endpoint string, username, password string) (*grpc.ClientConn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	creds := credentials.NewTLS(tlsConfig)
	return grpc.Dial(endpoint, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(basic.PerRPCCredentials(username, password)))
}

// GRPCConnectionWithToken returns a grpc.ClientConn using the given token to authenticate
func GRPCConnectionWithToken(endpoint, token string) (*grpc.ClientConn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	creds := credentials.NewTLS(tlsConfig)
	return grpc.Dial(endpoint, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(tokenbased.PerRPCCredentials(token)))
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
