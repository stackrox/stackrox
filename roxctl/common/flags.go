package common

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/grpc/alpn"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
	http1DowngradeClient "golang.stackrox.io/grpc-http1/client"
	"google.golang.org/grpc"
)

// GetGRPCConnection gets a grpc connection to Central with the correct auth
func GetGRPCConnection() (*grpc.ClientConn, error) {
	endpoint, usePlaintext, err := flags.EndpointAndPlaintextSetting()
	if err != nil {
		return nil, err
	}

	tlsOpts, err := tlsConfigOptsForCentral()
	if err != nil {
		return nil, err
	}

	opts := clientconn.Options{
		TLS: *tlsOpts,
	}

	var grpcDialOpts []grpc.DialOption

	if usePlaintext {
		if !flags.UseInsecure() {
			return nil, errors.New("plaintext connection mode must be used in conjunction with --insecure")
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
			return http1DowngradeClient.ConnectViaProxy(ctx, endpoint, tlsClientConf, http1DowngradeClient.ForceDowngrade(flags.ForceHTTP1()), http1DowngradeClient.ExtraH2ALPNs(alpn.PureGRPCALPNString), http1DowngradeClient.DialOpts(opts...))
		}
	} else if flags.ForceHTTP1() {
		return nil, errors.New("cannot force HTTP/1 mode if direct gRPC is enabled")
	}

	if err := checkAuthParameters(); err != nil {
		return nil, err
	}
	if flags.Password() != "" {
		opts.ConfigureBasicAuth(basic.DefaultUsername, flags.Password())
	} else {
		apiToken, err := retrieveAuthToken()
		if err != nil {
			printAuthHelp()
			return nil, err
		}
		if apiToken != "" {
			opts.ConfigureTokenAuth(apiToken)
		}
	}

	return clientconn.GRPCConnection(common.Context(), mtls.CentralSubject, endpoint, opts, grpcDialOpts...)
}
