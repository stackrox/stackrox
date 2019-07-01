package clientconn

import (
	"crypto/tls"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/grpc/authn/servicecerttoken"
	"github.com/stackrox/rox/pkg/grpc/authn/tokenbased"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// TLSConfig returns a TLS config that can be used to talk to the given server via MTLS.
func TLSConfig(server mtls.Subject, useClientCert bool) (*tls.Config, error) {
	rootCAs, err := verifier.TrustedCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "trusted pool")
	}

	conf := &tls.Config{
		ServerName: server.Hostname(),
		RootCAs:    rootCAs,
	}

	if useClientCert {
		clientCert, err := mtls.LeafCertificateFromFile()
		if err != nil {
			return nil, errors.Wrap(err, "client credentials")
		}
		conf.Certificates = []tls.Certificate{clientCert}
	}

	return conf, nil
}

type connectionOptions struct {
	useServiceCertToken bool
}

// ConnectionOption allows specifying additional options when establishing GRPC connections.
type ConnectionOption interface {
	apply(opts *connectionOptions) error
}

type connectOptFunc func(opts *connectionOptions) error

func (f connectOptFunc) apply(opts *connectionOptions) error {
	return f(opts)
}

// UseServiceCertToken specifies whether or not a `ServiceCert` token should be used.
func UseServiceCertToken(use bool) ConnectionOption {
	return connectOptFunc(func(opts *connectionOptions) error {
		opts.useServiceCertToken = use
		return nil
	})
}

// AuthenticatedGRPCConnection returns a grpc.ClientConn object that uses
// client certificates found on the local file system.
func AuthenticatedGRPCConnection(endpoint string, server mtls.Subject, extraConnOpts ...ConnectionOption) (conn *grpc.ClientConn, err error) {
	tlsConfig, err := TLSConfig(server, true)
	if err != nil {
		return nil, err
	}

	var connOpts connectionOptions
	for _, opt := range extraConnOpts {
		if err := opt.apply(&connOpts); err != nil {
			return nil, errors.Wrap(err, "failed to apply connection option")
		}
	}

	creds := credentials.NewTLS(tlsConfig)
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		keepAliveDialOption(),
	}

	if connOpts.useServiceCertToken {
		leafCert, err := mtls.LeafCertificateFromFile()
		if err != nil {
			return nil, errors.Wrap(err, "loading client certificate")
		}
		dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(servicecerttoken.NewServiceCertClientCreds(&leafCert)))
	}

	return grpc.Dial(endpoint, dialOpts...)
}

// GRPCConnectionWithBasicAuth returns a grpc.ClientConn using the given username/password to authenticate
// via basic auth.
func GRPCConnectionWithBasicAuth(endpoint string, serverName, username, password string) (*grpc.ClientConn, error) {
	return grpcConnectionWithPerRPCCreds(endpoint, serverName, basic.PerRPCCredentials(username, password))
}

// GRPCConnectionWithToken returns a grpc.ClientConn using the given token to authenticate
func GRPCConnectionWithToken(endpoint, serverName, token string) (*grpc.ClientConn, error) {
	return grpcConnectionWithPerRPCCreds(endpoint, serverName, tokenbased.PerRPCCredentials(token))
}

func grpcConnectionWithPerRPCCreds(endpoint string, serverName string, perRPCCreds credentials.PerRPCCredentials) (*grpc.ClientConn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         serverName,
	}
	creds := credentials.NewTLS(tlsConfig)
	return grpc.Dial(endpoint, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPCCreds))
}

// Parameters for keep alive.
func keepAliveDialOption() grpc.DialOption {
	// Since we are holding open a GRPC stream, enable keep alive.
	// Ping every minute of inactivity, and wait 30 seconds. Do this even when no streams are open (though
	// one should always be open with central.)
	params := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             30 * time.Second,
		PermitWithoutStream: true,
	}
	return grpc.WithKeepaliveParams(params)
}
