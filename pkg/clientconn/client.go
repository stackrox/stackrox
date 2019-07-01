package clientconn

import (
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/grpc/authn/servicecerttoken"
	"github.com/stackrox/rox/pkg/grpc/authn/tokenbased"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/netutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// TLSConfigOptions are options that modify the behavior of `TLSConfig`.
type TLSConfigOptions struct {
	UseClientCert bool
	ServerName    string
}

// verifyPeerCertificateFunc returns a function that can be used as the `VerifyPeerCertificate` callback of a
// tls.Config. It first tries to verify the peer certificate against the StackRox service CA (with a ServerName derived
// from server), and if that fails, tries to verify the peer certificate as a third-party certificate trusted by a
// system trust root.
func verifyPeerCertificateFunc(server mtls.Subject, serviceCA *x509.CertPool, serverName string) func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		var leafCert *x509.Certificate
		intermediates := x509.NewCertPool()
		for i, rawCert := range rawCerts {
			cert, err := x509.ParseCertificate(rawCert)
			if err != nil {
				return errors.Wrap(err, "could not parse peer certificate")
			}
			if i == 0 {
				leafCert = cert
			} else {
				intermediates.AddCert(cert)
			}
		}

		if leafCert == nil {
			return errors.New("no peer certificates provided")
		}

		// Try verifying StackRox Service Cert
		serviceVerifyOpts := x509.VerifyOptions{
			DNSName:       server.Hostname(),
			Intermediates: intermediates,
			Roots:         serviceCA,
		}

		verifyErrors := errorhelpers.NewErrorList("peer certificate validation failed")
		if _, err := leafCert.Verify(serviceVerifyOpts); err != nil {
			verifyErrors.AddError(err)
		} else {
			return nil
		}

		// Try verifying 3rd party cert.
		thirdPartyVerifyOpts := x509.VerifyOptions{
			DNSName:       serverName,
			Intermediates: intermediates,
			Roots:         nil, // use system roots
		}
		if _, err := leafCert.Verify(thirdPartyVerifyOpts); err != nil {
			verifyErrors.AddError(err)
		} else {
			return nil
		}

		return verifyErrors.ToError()
	}
}

// TLSConfig returns a TLS config that can be used to talk to the given server via mTLS.
func TLSConfig(server mtls.Subject, opts TLSConfigOptions) (*tls.Config, error) {
	serviceCA, err := verifier.TrustedCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "trusted pool")
	}

	serverName := opts.ServerName
	if serverName == "" {
		serverName = server.Hostname()
	}

	conf := &tls.Config{
		ServerName: serverName,
		RootCAs:    serviceCA,
	}

	if opts.UseClientCert {
		clientCert, err := mtls.LeafCertificateFromFile()
		if err != nil {
			return nil, errors.Wrap(err, "client credentials")
		}
		conf.Certificates = []tls.Certificate{clientCert}
	}

	if serverName != server.Hostname() {
		// Since we want to support verifying against both a StackRox Service CA or a System CA, we don't know the
		// ServerName against which to verify. While we could leave the `ServerName` field empty and only check the
		// `ServerName` of the verified chains passed to the `VerifyPeerCertificate` callback, this would have the
		// undesired side effect of not sending SNI in the handshake. Hence, skip the verification done by the tls
		// library altogether and do everything in our `VerifyPeerCertificate` callback.
		// Don't worry - this looks scary, but is actually not insecure; just slightly more flexible than what the
		// tls library supports natively.
		conf.InsecureSkipVerify = true
		conf.VerifyPeerCertificate = verifyPeerCertificateFunc(server, serviceCA, serverName)
	}

	// If the ServerName is an IP address, no SNI will be sent by the client. Because some SNI is always better than no
	// SNI, send the canonical hostname as the ServerName. Note that if `serverName` is an IP address, we will still
	// verify the peer certificate's IP SANs (if any) against this address.
	if netutil.IsIPAddress(conf.ServerName) {
		conf.ServerName = server.Hostname()
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
	var connOpts connectionOptions
	for _, opt := range extraConnOpts {
		if err := opt.apply(&connOpts); err != nil {
			return nil, errors.Wrap(err, "failed to apply connection option")
		}
	}

	host, _, _, err := netutil.ParseEndpoint(endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse endpoint %q", endpoint)
	}

	tlsConfig, err := TLSConfig(server, TLSConfigOptions{
		UseClientCert: true,
		ServerName:    host,
	})
	if err != nil {
		return nil, err
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
