package common

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/utils"
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

// DoHTTPRequestAndCheck200 does an http request to the provided path in Central,
// and passes through the remaining params. It checks that the returned status code is 200, and returns an error if it is not.
// The caller receives the http response object, which it is the caller's responsibility to close.
func DoHTTPRequestAndCheck200(path string, timeout time.Duration, method string, body io.Reader) (*http.Response, error) {
	url := GetURL(path)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	AddAuthToRequest(req)

	client := GetHTTPClient(timeout)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		defer utils.IgnoreError(resp.Body.Close)
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "Expected status code 200, but received %d. Additionally, there was an error reading the response", resp.StatusCode)
		}
		return nil, errors.Errorf("Expected status code 200, but received %d. Response Body: %s", resp.StatusCode, string(data))
	}

	return resp, nil
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
