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
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/flags"
)

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
	req, err := NewHTTPRequestWithAuth(method, path, body)
	if err != nil {
		return nil, err
	}

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

func getURL(path string) string {
	scheme := "https"
	if flags.UsePlaintext() {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s/%s", scheme, flags.Endpoint(), strings.TrimLeft(path, "/"))
}

// NewHTTPRequestWithAuth returns a new HTTP request, resolving the given path against the endpoint via `GetPath`, and
// injecting authorization headers into the request.
func NewHTTPRequestWithAuth(method string, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, getURL(path), body)
	if err != nil {
		return nil, err
	}
	if req.URL.Scheme != "https" && !flags.UseInsecure() {
		return nil, errors.Errorf("URL %v uses insecure scheme %q, use --insecure flags to enable sending credentials", req.URL, req.URL.Scheme)
	}
	AddAuthToRequest(req)

	return req, nil
}
