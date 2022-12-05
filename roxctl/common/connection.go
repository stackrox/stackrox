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
	"github.com/stackrox/rox/pkg/grpc/client/authn/basic"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/logger"
	http1DowngradeClient "golang.stackrox.io/grpc-http1/client"
	"google.golang.org/grpc"
)

// GetGRPCConnection gets a grpc connection to Central with the correct auth
func GetGRPCConnection(logger logger.Logger) (*grpc.ClientConn, error) {
	insecure := flags.UseInsecure()
	useDirectGRPC := flags.UseDirectGRPC()
	forceHTTP1 := flags.ForceHTTP1()

	endpoint, serverName, usePlaintext, err := ConnectNames()
	if err != nil {
		return nil, errors.Wrap(err, "could not get endpoint for gRPC connection")
	}

	opts, err := getAuthOpts(logger)
	if err != nil {
		return nil, err
	}

	return createGRPCConn(grpcConfig{
		usePlaintext:  usePlaintext,
		insecure:      insecure,
		opts:          opts,
		serverName:    serverName,
		useDirectGRPC: useDirectGRPC,
		forceHTTP1:    forceHTTP1,
		endpoint:      endpoint,
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
}

func createGRPCConn(c grpcConfig) (*grpc.ClientConn, error) {
	retryOpts := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffExponential(100 * time.Millisecond)),
		grpc_retry.WithMax(3),
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

func getAuthOpts(logger logger.Logger) (clientconn.Options, error) {
	tlsOpts, err := tlsConfigOptsForCentral(logger)
	if err != nil {
		return clientconn.Options{}, err
	}
	if err := checkAuthParameters(); err != nil {
		return clientconn.Options{}, err
	}

	opts := clientconn.Options{
		TLS: *tlsOpts,
	}

	password := flags.Password()
	if password != "" {
		opts.ConfigureBasicAuth(basic.DefaultUsername, password)
		return opts, nil
	}
	apiToken, err := retrieveAuthToken()
	if err != nil {
		printAuthHelp(logger)
		return clientconn.Options{}, err
	}
	if apiToken != "" {
		opts.ConfigureTokenAuth(apiToken)
	}
	return opts, nil
}
