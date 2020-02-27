package common

import (
	"net"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	http1DowngradeClient "github.com/stackrox/rox/pkg/grpc/http1downgrade/client"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
	"google.golang.org/grpc"
)

// GetGRPCConnection gets a grpc connection to Central with the correct auth
func GetGRPCConnection() (*grpc.ClientConn, error) {
	endpoint := flags.Endpoint()

	tlsOpts, err := tlsConfigOptsForCentral()
	if err != nil {
		return nil, err
	}

	opts := clientconn.Options{
		TLS: *tlsOpts,
	}

	var grpcDialOpts []grpc.DialOption

	if flags.UsePlaintext() {
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
		opts.DialTLS = http1DowngradeClient.ConnectViaProxy
	}

	if token := env.TokenEnv.Setting(); token != "" {
		opts.ConfigureTokenAuth(token)
	} else {
		opts.ConfigureBasicAuth(basic.DefaultUsername, flags.Password())
	}

	return clientconn.GRPCConnection(common.Context(), mtls.CentralSubject, endpoint, opts, grpcDialOpts...)
}
