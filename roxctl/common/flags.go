package common

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"google.golang.org/grpc"
)

var (
	password string
	endpoint string
)

// AddAuthFlags adds the endpoint to the base command
// This package provides the ability to take the global flags
func AddAuthFlags(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&password, "password", "p", "", "password for basic auth")
	c.PersistentFlags().StringVarP(&endpoint, "endpoint", "e", "localhost:8443", "endpoint for service to contact")
}

// GetGRPCConnection gets a grpc connection to Central with the correct auth
func GetGRPCConnection() (*grpc.ClientConn, error) {
	if token := env.TokenEnv.Setting(); token != "" {
		return clientconn.GRPCConnectionWithToken(endpoint, token)
	}
	return clientconn.GRPCConnectionWithBasicAuth(endpoint, basic.DefaultUsername, password)
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
		req.SetBasicAuth(basic.DefaultUsername, password)
	}
}

// GetURL adds the endpoint to the passed path
func GetURL(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("https://%s"+path, endpoint)
}
