package common

import (
	"fmt"
	"net"
	"os"
	"strings"

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

const userHelpLiteralToken = `There is no token in file %q. The token file should only contain a single authentication token.
To provide a token value directly, set the ROX_API_TOKEN environment variable.
`

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
		opts.DialTLS = http1DowngradeClient.ConnectViaProxy
	}

	// Try to retrieve API token. First via --token-file parameter and then from the environment.
	// Sets apiToken on success.
	var apiToken string
	if tokenFile := flags.APITokenFile(); tokenFile != "" {
		// Error out if --token-file and --password is present on the command line.
		if flags.Password() != "" {
			return nil, errors.New("Cannot use password- and token-based authentication at the same time")
		}

		apiToken, err = flags.ReadTokenFromFile(tokenFile)
		if err != nil {
			if !strings.Contains(tokenFile, "/") {
				// Specified token file looks somewhat like a literal token, try to help the user.
				fmt.Fprintf(os.Stderr, userHelpLiteralToken, tokenFile)
			}
			return nil, err
		}
	} else if token := env.TokenEnv.Setting(); token != "" {
		apiToken = token
	}

	if flags.Password() != "" {
		opts.ConfigureBasicAuth(basic.DefaultUsername, flags.Password())
	} else if apiToken != "" {
		opts.ConfigureTokenAuth(apiToken)
	}

	return clientconn.GRPCConnection(common.Context(), mtls.CentralSubject, endpoint, opts, grpcDialOpts...)
}
