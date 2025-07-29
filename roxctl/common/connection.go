package common

import (
	"context"
	"crypto/tls"
	"net"
	"strings"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/alpn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/auth"
	"github.com/stackrox/rox/roxctl/common/flags"
	http1DowngradeClient "golang.stackrox.io/grpc-http1/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCOption encodes behavior of a gRPC connection.
type GRPCOption func(*grpcConfig)

// WithRetryTimeout sets a retry timeout for the gRPC connection.
func WithRetryTimeout(timeout time.Duration) GRPCOption {
	return func(config *grpcConfig) {
		config.retryTimeout = timeout
	}
}

// GetGRPCConnection gets a grpc connection to Central with the correct auth
func GetGRPCConnection(am auth.Method, connectionOpts ...GRPCOption) (*grpc.ClientConn, error) {
	endpoint, serverName, usePlaintext, err := ConnectNames()
	if err != nil {
		return nil, errors.Wrap(err, "could not get endpoint for gRPC connection")
	}
	perRPCCreds, err := am.GetCredentials(endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "obtaining auth information for %s", endpoint)
	}
	clientOpts, err := getClientOpts()
	if err != nil {
		return nil, err
	}
	clientOpts.PerRPCCreds = perRPCCreds

	config := grpcConfig{
		usePlaintext:  usePlaintext,
		insecure:      flags.UseInsecure(),
		opts:          clientOpts,
		serverName:    serverName,
		useDirectGRPC: flags.UseDirectGRPC(),
		forceHTTP1:    flags.ForceHTTP1(),
		endpoint:      endpoint,
	}

	for _, opt := range connectionOpts {
		opt(&config)
	}

	return createGRPCConn(config)
}

type grpcConfig struct {
	usePlaintext  bool
	insecure      bool
	opts          clientconn.Options
	serverName    string
	useDirectGRPC bool
	forceHTTP1    bool
	endpoint      string
	retryTimeout  time.Duration
}

func makeCtxWithCommandHeader(ctx context.Context) context.Context {
	md := metadata.New(nil)
	setCustomHeaders(md.Set)
	return metadata.NewOutgoingContext(ctx, md)
}

// addCommandHeaderUnaryInterceptor adds the roxctl command header to all unary requests.
func addCommandHeaderUnaryInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return invoker(makeCtxWithCommandHeader(ctx), method, req, reply, cc, opts...)
}

// addCommandHeaderUnaryInterceptor adds the roxctl command header to all stream requests.
func addCommandHeaderStreamInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return streamer(makeCtxWithCommandHeader(ctx), desc, cc, method, opts...)
}

func shouldRetry(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}
	if strings.Contains(err.Error(), "x509: certificate") {
		return false
	}
	if grpcErr, ok := status.FromError(err); ok {
		code := grpcErr.Code()
		if code != codes.Unavailable && code != codes.ResourceExhausted {
			return false
		}
	}
	return true
}

func createGRPCConn(c grpcConfig) (*grpc.ClientConn, error) {
	const initialBackoffDuration = 100 * time.Millisecond
	retryOpts := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffExponential(initialBackoffDuration)),
		// First retry after 100ms, last retry after 51.2s.
		grpc_retry.WithMax(10),
		grpc_retry.WithPerRetryTimeout(c.retryTimeout),
		grpc_retry.WithRetriable(shouldRetry),
	}

	grpcDialOpts := []grpc.DialOption{
		grpc.WithChainStreamInterceptor(
			addCommandHeaderStreamInterceptor,
			grpc_retry.StreamClientInterceptor(retryOpts...)),
		grpc.WithChainUnaryInterceptor(
			addCommandHeaderUnaryInterceptor,
			grpc_retry.UnaryClientInterceptor(retryOpts...)),
	}

	if c.usePlaintext {
		if !c.insecure {
			return nil, errox.InvalidArgs.New("plaintext connection mode must be used in conjunction with --insecure")
		}
		c.opts.InsecureNoTLS = true
		c.opts.InsecureAllowCredsViaPlaintext = true

		// Set the server name as the authority since we don't have SNI (don't set it for IP addresses).
		if c.serverName != "" && net.ParseIP(c.serverName) == nil {
			grpcDialOpts = append(grpcDialOpts, grpc.WithAuthority(c.serverName))
		}
	} else if c.opts.TLS.DialContext != nil {
		grpcDialOpts = append(grpcDialOpts, grpc.WithContextDialer(c.opts.TLS.DialContext))
	}

	if !c.useDirectGRPC {
		c.opts.DialTLS = func(ctx context.Context, endpoint string, tlsClientConf *tls.Config, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
			proxy, proxyErr := http1DowngradeClient.ConnectViaProxy(
				ctx,
				endpoint,
				tlsClientConf,
				http1DowngradeClient.ForceDowngrade(c.forceHTTP1),
				http1DowngradeClient.ExtraH2ALPNs(alpn.PureGRPCALPNString),
				http1DowngradeClient.DialOpts(opts...),
			)
			return proxy, errors.Wrap(proxyErr, "could not connect via proxy")
		}
	} else if c.forceHTTP1 {
		return nil, errox.InvalidArgs.New("cannot force HTTP/1 mode if direct gRPC is enabled")
	}

	connection, err := clientconn.GRPCConnection(common.Context(), mtls.CentralSubject, c.endpoint, c.opts, grpcDialOpts...)
	return connection, errors.WithStack(err)
}

func getClientOpts() (clientconn.Options, error) {
	tlsOpts, err := tlsConfigOptsForCentral()
	if err != nil {
		return clientconn.Options{}, err
	}
	opts := clientconn.Options{
		TLS: *tlsOpts,
	}
	return opts, nil
}
