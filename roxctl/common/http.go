package common

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/flags"
	"golang.org/x/net/http2"
)

var (
	http1NextProtos = []string{"http/1.1", "http/1.0"}
)

// GetHTTPClient gets a client with the correct config
func GetHTTPClient(timeout time.Duration) (*http.Client, error) {
	tlsConf, err := tlsConfigForCentral()
	if err != nil {
		return nil, errors.Wrap(err, "instantiating TLS configuration for central")
	}
	transport := &http.Transport{
		TLSClientConfig: tlsConf,
	}
	if flags.ForceHTTP1() {
		transport.TLSClientConfig.NextProtos = http1NextProtos
	} else {
		// There's no reason to not use HTTP/2, but we don't go out of our way to do so.
		if err := http2.ConfigureTransport(transport); err != nil {
			transport.TLSClientConfig.NextProtos = http1NextProtos
		}
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	return client, nil
}

// DoHTTPRequestAndCheck200 does an http request to the provided path in Central,
// and passes through the remaining params. It checks that the returned status code is 200, and returns an error if it is not.
// The caller receives the http response object, which it is the caller's responsibility to close.
func DoHTTPRequestAndCheck200(path string, timeout time.Duration, method string, body io.Reader) (*http.Response, error) {
	req, err := NewHTTPRequestWithAuth(method, path, body)
	if err != nil {
		return nil, err
	}

	client, err := GetHTTPClient(timeout)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		defer utils.IgnoreError(resp.Body.Close)
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "Expected status code 200, but received %d. Additionally, there was an error reading the response", resp.StatusCode)
		}
		return nil, errors.Errorf("Expected status code 200, but received %d. Response Body: %s", resp.StatusCode, string(data))
	}

	return resp, nil
}

// AddAuthToRequest adds the correct auth to the request
func AddAuthToRequest(req *http.Request) error {
	token, err := RetrieveAuthToken()
	if err != nil {
		return errors.Wrap(err, "Failed to enrich HTTP request with authentication information")
	}

	if flags.Password() != "" {
		req.SetBasicAuth(basic.DefaultUsername, flags.Password())
	} else if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	return nil
}

func getURL(path string) (string, error) {
	endpoint, usePlaintext, err := flags.EndpointAndPlaintextSetting()
	if err != nil {
		return "", err
	}
	scheme := "https"
	if usePlaintext {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s/%s", scheme, endpoint, strings.TrimLeft(path, "/")), nil
}

// NewHTTPRequestWithAuth returns a new HTTP request, resolving the given path against the endpoint via `GetPath`, and
// injecting authorization headers into the request.
func NewHTTPRequestWithAuth(method string, path string, body io.Reader) (*http.Request, error) {
	reqURL, err := getURL(path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, err
	}
	if flags.ForceHTTP1() {
		req.ProtoMajor, req.ProtoMinor, req.Proto = 1, 1, "HTTP/1.1"
	}

	if req.URL.Scheme != "https" && !flags.UseInsecure() {
		return nil, errors.Errorf("URL %v uses insecure scheme %q, use --insecure flags to enable sending credentials", req.URL, req.URL.Scheme)
	}
	err = AddAuthToRequest(req)
	if err != nil {
		return nil, err
	}

	return req, nil
}
