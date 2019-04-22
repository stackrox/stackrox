package common

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

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

	if token := env.TokenEnv.Setting(); token != "" {
		return clientconn.GRPCConnectionWithToken(endpoint, serverName, token)
	}
	return clientconn.GRPCConnectionWithBasicAuth(endpoint, serverName, basic.DefaultUsername, flags.Password())
}

// GetHTTPClient gets a client with the correct config
func GetHTTPClient(timeout time.Duration) *http.Client {
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	return client
}

// AddAuthToRequest adds the correct auth to the request
func AddAuthToRequest(req *http.Request) {
	if token := env.TokenEnv.Setting(); token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	} else {
		req.SetBasicAuth(basic.DefaultUsername, flags.Password())
	}
}

// GetURL adds the endpoint to the passed path
func GetURL(path string) string {
	return fmt.Sprintf("https://%s/%s", flags.Endpoint(), strings.TrimLeft(path, "/"))
}
