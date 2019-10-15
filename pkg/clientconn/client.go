package clientconn

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/alpn"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/grpc/authn/servicecerttoken"
	"github.com/stackrox/rox/pkg/grpc/authn/tokenbased"
	"github.com/stackrox/rox/pkg/grpc/util"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/netutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

var (
	// NextProtos are the ALPN protos to use for gRPC connections.
	NextProtos = []string{alpn.PureGRPCALPNString, "h2", "http/1.1"}
	// NextProtosNoPureGRPC are the ALPN protos to use if the connection needs to support plain HTTP in addition to
	// only gRPC calls.
	NextProtosNoPureGRPC = []string{"h2", "http/1.1"}

	log = logging.LoggerForModule()
)

// Options specifies options for establishing a gRPC client connection.
type Options struct {
	InsecureNoTLS bool
	TLS           TLSConfigOptions

	InsecureAllowCredsViaPlaintext bool
	PerRPCCreds                    credentials.PerRPCCredentials

	DialTLS DialTLSFunc
}

func (o *Options) dialTLSFunc() DialTLSFunc {
	if o.DialTLS != nil {
		return o.DialTLS
	}
	return DialTLS
}

func (o *Options) tlsConfig(server mtls.Subject) (*tls.Config, error) {
	return TLSConfig(server, o.TLS)
}

// ConfigureBasicAuth configures this client connection to authenticate via basic authentication.
func (o *Options) ConfigureBasicAuth(username, password string) {
	o.PerRPCCreds = basic.PerRPCCredentials(username, password)
}

// ConfigureTokenAuth configures this client connection to authenticate via token-based authentication.
func (o *Options) ConfigureTokenAuth(token string) {
	o.PerRPCCreds = tokenbased.PerRPCCredentials(token)
}

// TLSConfigOptions are options that modify the behavior of `TLSConfig`.
type TLSConfigOptions struct {
	UseClientCert      bool
	ServerName         string
	InsecureSkipVerify bool
	CustomCertVerifier TLSCertVerifier
	RootCAs            *x509.CertPool

	GRPCOnly bool
}

// TLSConfig returns a TLS config that can be used to talk to the given server via mTLS.
func TLSConfig(server mtls.Subject, opts TLSConfigOptions) (*tls.Config, error) {
	serverName := opts.ServerName
	if serverName == "" {
		serverName = server.Hostname()
	}

	nextProtos := NextProtos
	if !opts.GRPCOnly {
		nextProtos = NextProtosNoPureGRPC // no pure gRPC
	}

	conf := &tls.Config{
		ServerName: serverName,
		NextProtos: nextProtos,
		RootCAs:    opts.RootCAs,
	}

	if opts.UseClientCert {
		clientCert, err := mtls.LeafCertificateFromFile()
		if err != nil {
			return nil, errors.Wrap(err, "client credentials")
		}
		conf.Certificates = []tls.Certificate{clientCert}
	}

	customVerifier := opts.CustomCertVerifier
	if !opts.InsecureSkipVerify && customVerifier == nil {
		// Try verifying the remote certificate as a StackRox service certificate (locate the service CA cert in the
		// custom root CA pool, or the standard mTLS root CA certificate location).
		serviceCA := opts.RootCAs
		if serviceCA == nil {
			var err error
			serviceCA, err = verifier.TrustedCertPool()
			if err != nil {
				// Not an error - this code path is invoked from `roxctl` as well, in which case we don't expect a
				// `/run/secrets/stackrox.io/...` directory structure to exist.
				serviceCA = nil
			}
		}

		if serviceCA != nil {
			customVerifier = &serviceCertFallbackVerifier{
				serviceCAs: serviceCA,
				subject:    server,
			}
		}
	} else if opts.InsecureSkipVerify {
		conf.InsecureSkipVerify = true
	}

	if customVerifier != nil {
		conf.VerifyPeerCertificate = verifyPeerCertFunc(conf, customVerifier)
		conf.InsecureSkipVerify = true
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
	dialTLSFunc         DialTLSFunc
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

// UseDialTLSFunc uses the given connection function for dialing.
func UseDialTLSFunc(fn DialTLSFunc) ConnectionOption {
	return connectOptFunc(func(opts *connectionOptions) error {
		opts.dialTLSFunc = fn
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

	clientConnOpts := Options{
		TLS: TLSConfigOptions{
			UseClientCert: true,
			ServerName:    host,
			GRPCOnly:      true,
		},
		DialTLS: connOpts.dialTLSFunc,
	}

	if connOpts.useServiceCertToken {
		leafCert, err := mtls.LeafCertificateFromFile()
		if err != nil {
			return nil, errors.Wrap(err, "loading client certificate")
		}
		clientConnOpts.PerRPCCreds = servicecerttoken.NewServiceCertClientCreds(&leafCert)
	}

	return GRPCConnection(context.Background(), server, endpoint, clientConnOpts, keepAliveDialOption())
}

// GRPCConnection establishes a gRPC connection to the given server, using the given connection options.
func GRPCConnection(dialCtx context.Context, server mtls.Subject, endpoint string, clientConnOpts Options, dialOpts ...grpc.DialOption) (*grpc.ClientConn, error) {
	allDialOpts := make([]grpc.DialOption, 0, len(dialOpts)+2)

	clientConnOpts.TLS.GRPCOnly = true

	var tlsConf *tls.Config
	if !clientConnOpts.InsecureNoTLS {
		var err error
		tlsConf, err = clientConnOpts.tlsConfig(server)
		if err != nil {
			return nil, errors.Wrap(err, "instantiating TLS config")
		}
	} else {
		allDialOpts = append(allDialOpts, grpc.WithInsecure())
	}

	if perRPCCreds := clientConnOpts.PerRPCCreds; perRPCCreds != nil {
		if clientConnOpts.InsecureNoTLS && clientConnOpts.InsecureAllowCredsViaPlaintext {
			perRPCCreds = util.ForceInsecureCreds(perRPCCreds)
		}
		allDialOpts = append(allDialOpts, grpc.WithPerRPCCredentials(perRPCCreds))
	}
	allDialOpts = append(allDialOpts, dialOpts...)
	return clientConnOpts.dialTLSFunc()(dialCtx, endpoint, tlsConf, allDialOpts...)
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
