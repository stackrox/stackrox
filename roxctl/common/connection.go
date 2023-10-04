package common

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/alpn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/auth"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/logger"
	http1DowngradeClient "golang.stackrox.io/grpc-http1/client"
	"google.golang.org/grpc"
)

// GetGRPCConnection gets a grpc connection to Central with the correct auth
func GetGRPCConnection(am auth.Method, logger logger.Logger, retryTimeout time.Duration) (*grpc.ClientConn, error) {
	endpoint, serverName, usePlaintext, err := ConnectNames()
	if err != nil {
		return nil, errors.Wrap(err, "could not get endpoint for gRPC connection")
	}
	perRPCCreds, err := am.GetCredentials(endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "obtaining auth information for %s", endpoint)
	}
	opts, err := getOpts(logger)
	if err != nil {
		return nil, err
	}
	opts.PerRPCCreds = perRPCCreds

	return createGRPCConn(grpcConfig{
		usePlaintext:  usePlaintext,
		insecure:      flags.UseInsecure(),
		opts:          opts,
		serverName:    serverName,
		useDirectGRPC: flags.UseDirectGRPC(),
		forceHTTP1:    flags.ForceHTTP1(),
		endpoint:      endpoint,
		retryTimeout:  retryTimeout,
	})
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

func createGRPCConn(c grpcConfig) (*grpc.ClientConn, error) {
	const initialBackoffDuration = 100 * time.Millisecond
	retryOpts := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffExponential(initialBackoffDuration)),
		// First retry after 100ms, last retry after 51.2s.
		grpc_retry.WithMax(10),
		grpc_retry.WithPerRetryTimeout(c.retryTimeout),
	}

	grpcDialOpts := []grpc.DialOption{
		grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor(retryOpts...)),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(retryOpts...)),
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

func getOpts(logger logger.Logger) (clientconn.Options, error) {
	tlsOpts, err := tlsConfigOptsForCentral(logger)
	if err != nil {
		return clientconn.Options{}, err
	}
	opts := clientconn.Options{
		TLS: *tlsOpts,
	}
	return opts, nil
}
