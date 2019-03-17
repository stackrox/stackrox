package common

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/roxctl/common/flags"
	"google.golang.org/grpc"
)

// GetGRPCConnection gets a grpc connection to Central with the correct auth
func GetGRPCConnection() (*grpc.ClientConn, error) {
	if token := env.TokenEnv.Setting(); token != "" {
		return clientconn.GRPCConnectionWithToken(flags.Endpoint(), token)
	}
	return clientconn.GRPCConnectionWithBasicAuth(flags.Endpoint(), basic.DefaultUsername, flags.Password())
}

// GetHTTPClientWithTimeout gets a client with the correct config
func GetHTTPClientWithTimeout(timeout time.Duration) *http.Client {
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

// GetHTTPClient returns an http client using the timeout set by the flag.
func GetHTTPClient() *http.Client {
	return GetHTTPClientWithTimeout(flags.Timeout())
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
