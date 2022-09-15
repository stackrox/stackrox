package common

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

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
func GetGRPCConnection(authMethod auth.Method, logger logger.Logger) (*grpc.ClientConn, error) {
	endpoint, usePlaintext, err := flags.EndpointAndPlaintextSetting()
	if err != nil {
		return nil, errors.Wrap(err, "could not get endpoint for gRPC connection")
	}

	tlsOpts, err := tlsConfigOptsForCentral(logger)
	if err != nil {
		return nil, err
	}

	opts := clientconn.Options{
		TLS: *tlsOpts,
	}

	var grpcDialOpts []grpc.DialOption

	if usePlaintext {
		if !flags.UseInsecure() {
			return nil, errox.InvalidArgs.New("plaintext connection mode must be used in conjunction with --insecure")
		}
		opts.InsecureNoTLS = true
		opts.InsecureAllowCredsViaPlaintext = true

		// Set the server name as the authority since we don't have SNI (don't set it for IP addresses).
		_, serverName, _ := ConnectNames()
		if serverName != "" && net.ParseIP(serverName) == nil {
			grpcDialOpts = append(grpcDialOpts, grpc.WithAuthority(serverName))
		}
	}

	if !flags.UseDirectGRPC() {
		opts.DialTLS = func(ctx context.Context, endpoint string, tlsClientConf *tls.Config, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
			proxy, proxyErr := http1DowngradeClient.ConnectViaProxy(
				ctx,
				endpoint,
				tlsClientConf,
				http1DowngradeClient.ForceDowngrade(flags.ForceHTTP1()),
				http1DowngradeClient.ExtraH2ALPNs(alpn.PureGRPCALPNString),
				http1DowngradeClient.DialOpts(opts...),
			)
			return proxy, errors.Wrap(proxyErr, "could not connect via proxy")
		}
	} else if flags.ForceHTTP1() {
		return nil, errox.InvalidArgs.New("cannot force HTTP/1 mode if direct gRPC is enabled")
	}

	scheme := "https"
	if usePlaintext {
		scheme = "http"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, endpoint)

	creds, err := authMethod.GetCreds(baseURL)
	if err != nil {
		return nil, errors.Wrapf(err, "could not obtain credentials for %s", baseURL)
	}
	opts.PerRPCCreds = creds

	connection, err := clientconn.GRPCConnection(common.Context(), mtls.CentralSubject, endpoint, opts, grpcDialOpts...)
	return connection, errors.WithStack(err)
}
