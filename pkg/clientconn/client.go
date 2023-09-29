package clientconn

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/alpn"
	"github.com/stackrox/rox/pkg/grpc/client/authn/basic"
	"github.com/stackrox/rox/pkg/grpc/client/authn/servicecerttoken"
	"github.com/stackrox/rox/pkg/grpc/client/authn/tokenbased"
	"github.com/stackrox/rox/pkg/grpc/util"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/tlscheck"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// UseClientCertSetting controls whether a client certificate should be used for the connection.
type UseClientCertSetting int

const (
	// DontUseClientCert will never attempt to use the client certificate for the connection.
	DontUseClientCert UseClientCertSetting = iota
	// UseClientCertIfAvailable will attempt to load the client certificate, but will not fail
	// with an error if the client certificate is unusable.
	UseClientCertIfAvailable
	// MustUseClientCert will fail the connection with an error if the client cert cannot be loaded.
	MustUseClientCert
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

	MaxMsgRecvSize int
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
	UseClientCert      UseClientCertSetting
	ServerName         string
	InsecureSkipVerify bool
	CustomCertVerifier tlscheck.TLSCertVerifier
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

	if opts.UseClientCert != DontUseClientCert {
		clientCert, err := mtls.LeafCertificateFromFile()
		if err != nil {
			if opts.UseClientCert == MustUseClientCert {
				return nil, errors.Wrap(err, "client credentials")
			}
			log.Warnf("Failed to load client certificate for TLS connection: %v", err)
		} else {
			conf.Certificates = []tls.Certificate{clientCert}
		}
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
		conf.VerifyPeerCertificate = tlscheck.VerifyPeerCertFunc(conf, customVerifier)
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
	useInsecureNoTLS    bool
	dialTLSFunc         DialTLSFunc
	rootCAs             *x509.CertPool
	maxMsgRecvSize      int
}

// ConnectionOption allows specifying additional options when establishing GRPC connections.
type ConnectionOption interface {
	apply(opts *connectionOptions) error
}

type connectOptFunc func(opts *connectionOptions) error

func (f connectOptFunc) apply(opts *connectionOptions) error {
	return f(opts)
}

// AddRootCAs adds new root certificates to the root CA cert pool for the gRPC connection
func AddRootCAs(certs ...*x509.Certificate) ConnectionOption {
	return connectOptFunc(func(opts *connectionOptions) error {
		if opts.rootCAs == nil {
			pool, err := verifier.SystemCertPool()
			if err != nil {
				return errors.Wrap(err, "Reading system certs")
			}
			opts.rootCAs = pool
		}

		for _, c := range certs {
			opts.rootCAs.AddCert(c)
		}
		return nil
	})
}

// MaxMsgReceiveSize overrides the default 4MB max receive size for gRPC client.
func MaxMsgReceiveSize(size int) ConnectionOption {
	return connectOptFunc(func(opts *connectionOptions) error {
		opts.maxMsgRecvSize = size
		return nil
	})
}

// UseServiceCertToken specifies whether a `ServiceCert` token should be used.
func UseServiceCertToken(use bool) ConnectionOption {
	return connectOptFunc(func(opts *connectionOptions) error {
		opts.useServiceCertToken = use
		return nil
	})
}

// UseInsecureNoTLS specifies whether to use insecure, non-TLS connections.
func UseInsecureNoTLS(use bool) ConnectionOption {
	return connectOptFunc(func(opts *connectionOptions) error {
		opts.useInsecureNoTLS = use
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

// OptionsForEndpoint returns an Options struct to be used with the given endpoint.
func OptionsForEndpoint(endpoint string, extraConnOpts ...ConnectionOption) (Options, error) {
	var connOpts connectionOptions
	for _, opt := range extraConnOpts {
		if err := opt.apply(&connOpts); err != nil {
			return Options{}, errors.Wrap(err, "failed to apply connection option")
		}
	}

	host, _, _, err := netutil.ParseEndpoint(endpoint)
	if err != nil {
		return Options{}, errors.Wrapf(err, "could not parse endpoint %q", endpoint)
	}

	clientConnOpts := Options{
		InsecureNoTLS: connOpts.useInsecureNoTLS,
		TLS: TLSConfigOptions{
			UseClientCert: MustUseClientCert,
			ServerName:    host,
			RootCAs:       connOpts.rootCAs,
		},
		DialTLS: connOpts.dialTLSFunc,
	}

	if connOpts.useServiceCertToken {
		leafCert, err := mtls.LeafCertificateFromFile()
		if err != nil {
			return Options{}, errors.Wrap(err, "loading client certificate")
		}
		clientConnOpts.PerRPCCreds = servicecerttoken.NewServiceCertClientCreds(&leafCert)
	}

	clientConnOpts.MaxMsgRecvSize = connOpts.maxMsgRecvSize

	return clientConnOpts, nil
}

// AuthenticatedGRPCConnection returns a grpc.ClientConn object that uses
// client certificates found on the local file system.
func AuthenticatedGRPCConnection(endpoint string, server mtls.Subject, extraConnOpts ...ConnectionOption) (conn *grpc.ClientConn, err error) {
	if strings.HasPrefix(endpoint, "ws://") || strings.HasPrefix(endpoint, "wss://") {
		_, endpoint = stringutils.Split2(endpoint, "://")
		extraConnOpts = append(extraConnOpts, UseDialTLSFunc(DialTLSWebSocket))
	}
	clientConnOpts, err := OptionsForEndpoint(endpoint, extraConnOpts...)
	if err != nil {
		return nil, err
	}

	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts, keepAliveDialOption())
	if clientConnOpts.MaxMsgRecvSize > 0 {
		dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(clientConnOpts.MaxMsgRecvSize)))
	}

	return GRPCConnection(context.Background(), server, endpoint, clientConnOpts, dialOpts...)
}

// HTTPTransport returns a RoundTripper for talking to the specified endpoint. The RoundTripper accepts requests with
// URLs that omit scheme and host; however, if scheme and/or host are specified, they MUST match "http" or "https"
// for the scheme (depending on TLS config)
func HTTPTransport(server mtls.Subject, endpoint string, clientConnOpts Options, baseTransport *http.Transport) (http.RoundTripper, error) {
	if clientConnOpts.DialTLS != nil {
		return nil, errors.New("custom TLS dialer is not supported for HTTP transport")
	}

	clientConnOpts.TLS.GRPCOnly = false

	var tlsConf *tls.Config
	var scheme string
	if !clientConnOpts.InsecureNoTLS {
		var err error
		tlsConf, err = clientConnOpts.tlsConfig(server)
		if err != nil {
			return nil, errors.Wrap(err, "instantiating TLS config")
		}
		scheme = "https"
	} else {
		scheme = "http"
	}

	var transport *http.Transport
	if baseTransport != nil {
		transport = baseTransport.Clone()
	} else {
		transport = httputil.DefaultTransport()
	}

	transport.TLSClientConfig = tlsConf
	if err := http2.ConfigureTransport(transport); err != nil {
		log.Warnf("Failed to configure HTTP/2 transport for talking to %v @ %s: %v", server.ServiceType, endpoint, err)
	}

	perRPCCreds := clientConnOpts.PerRPCCreds
	if perRPCCreds != nil {
		if clientConnOpts.InsecureNoTLS && clientConnOpts.InsecureAllowCredsViaPlaintext {
			perRPCCreds = util.ForceInsecureCreds(perRPCCreds)
		}
	}

	roundTripFunc := func(req *http.Request) (*http.Response, error) {
		modReq := req.Clone(req.Context())
		if modReq.URL.Scheme == "" {
			modReq.URL.Scheme = scheme
		} else if modReq.URL.Scheme != scheme {
			return nil, errors.Errorf("unexpected scheme %q, expected %q", modReq.URL.Scheme, scheme)
		}

		if modReq.URL.Host == "" {
			modReq.URL.Host = endpoint
		} else if modReq.URL.Host != endpoint {
			return nil, errors.Errorf("unexpected host %q, expected %q", modReq.URL.Host, endpoint)
		}

		// If there are per-RPC credentials configured, inject the respective metadata into the request header
		// (in case the per-RPC credentials require transport security, only do so if the target URL is using
		// the secure `https` scheme).
		if perRPCCreds != nil && (!perRPCCreds.RequireTransportSecurity() || modReq.URL.Scheme == "https") {
			authMD, err := perRPCCreds.GetRequestMetadata(modReq.Context(), modReq.URL.String())
			if err != nil {
				return nil, errors.Wrap(err, "retrieving authentication metadata")
			}
			for k, v := range authMD {
				modReq.Header.Add(k, v)
			}
		}

		return transport.RoundTrip(modReq)
	}

	return httputil.RoundTripperFunc(roundTripFunc), nil
}

// AuthenticatedHTTPTransport creates an HTTP transport for talking to the given service at the specified endpoint.
// The transport accepts URL without a schema and a host; however, if provided, they must match the expected values.
func AuthenticatedHTTPTransport(endpoint string, server mtls.Subject, baseTransport *http.Transport, extraConnOpts ...ConnectionOption) (http.RoundTripper, error) {
	if strings.HasPrefix(endpoint, "ws://") || strings.HasPrefix(endpoint, "wss://") {
		_, endpoint = stringutils.Split2(endpoint, "://")
		// No need to add the WebSocket TLS Dialer since this is not gRPC.
	}
	clientConnOpts, err := OptionsForEndpoint(endpoint, extraConnOpts...)
	if err != nil {
		return nil, err
	}

	return HTTPTransport(server, endpoint, clientConnOpts, baseTransport)
}

// GRPCConnection establishes a gRPC connection to the given server, using the given connection options.
func GRPCConnection(dialCtx context.Context, server mtls.Subject, endpoint string, clientConnOpts Options, dialOpts ...grpc.DialOption) (*grpc.ClientConn, error) {
	allDialOpts := make([]grpc.DialOption, 0, len(dialOpts)+3)

	clientConnOpts.TLS.GRPCOnly = true

	var tlsConf *tls.Config
	if !clientConnOpts.InsecureNoTLS {
		var err error
		tlsConf, err = clientConnOpts.tlsConfig(server)
		if err != nil {
			return nil, errors.Wrap(err, "instantiating TLS config")
		}
	} else {
		allDialOpts = append(allDialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if perRPCCreds := clientConnOpts.PerRPCCreds; perRPCCreds != nil {
		if clientConnOpts.InsecureNoTLS && clientConnOpts.InsecureAllowCredsViaPlaintext {
			perRPCCreds = util.ForceInsecureCreds(perRPCCreds)
		}
		allDialOpts = append(allDialOpts, grpc.WithPerRPCCredentials(perRPCCreds))
	}
	allDialOpts = append(allDialOpts, dialOpts...)
	allDialOpts = append(allDialOpts, grpc.WithUserAgent(GetUserAgent()))
	return clientConnOpts.dialTLSFunc()(dialCtx, endpoint, tlsConf, allDialOpts...)
}

// NewHTTPClient creates an HTTP client for the given service using the client
// certificate of the calling service.
// When specifying the url.URL for the *http.Request for the returned *http.Client to complete,
// there is no need to specify the Host nor Scheme; however,
// if provided, they both must match the expected values.
// See AuthenticatedHTTPTransport for more information.
func NewHTTPClient(serviceIdentity mtls.Subject, serviceEndpoint string, timeout time.Duration) (*http.Client, error) {
	transport, err := AuthenticatedHTTPTransport(
		serviceEndpoint, serviceIdentity, nil, UseServiceCertToken(true))
	if err != nil {
		return nil, errors.Wrap(err, "creating http transport")
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
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
