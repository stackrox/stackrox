package common

import (
	http1DowngradeClient "github.com/stackrox/rox/pkg/grpc/http1downgrade/client"
	"github.com/stackrox/rox/pkg/mtls"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/roxctl/common/flags"
	"google.golang.org/grpc"
)

// GetGRPCConnection gets a grpc connection to Central with the correct auth
func GetGRPCConnection() (*grpc.ClientConn, error) {
	endpoint := flags.Endpoint()
	serverName := flags.ServerName()
	if serverName == "" {
		var err error
		serverName, _, _, err = netutil.ParseEndpoint(endpoint)
		if err != nil {
			return nil, errors.Wrap(err, "parsing central endpoint")
		}
	}

	opts := clientconn.Options{
		TLS: clientconn.TLSConfigOptions{
			ServerName:         serverName,
			InsecureSkipVerify: true,
		},
	}
	if flags.UsePlaintext() {
		if !flags.UseInsecure() {
			return nil, errors.New("plaintext connection mode must be used in conjunction with --insecure")
		}
		opts.InsecureNoTLS = true
		opts.InsecureAllowCredsViaPlaintext = true
	}

	if !flags.UseDirectGRPC() {
		opts.DialTLS = http1DowngradeClient.ConnectViaProxy
	}

	if token := env.TokenEnv.Setting(); token != "" {
		opts.ConfigureTokenAuth(token)
	} else {
		opts.ConfigureBasicAuth(basic.DefaultUsername, flags.Password())
	}

	return clientconn.GRPCConnection(Context(), mtls.CentralSubject, endpoint, opts)
}
