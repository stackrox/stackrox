package common

import (
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
	"golang.org/x/net/http2"
)

// RoxctlHTTPClient abstracts all HTTP-related functionalities required within roxctl
type RoxctlHTTPClient interface {
	DoReqAndVerifyStatusCode(path string, method string, code int, body io.Reader) (*http.Response, error)
	NewReq(method string, path string, body io.Reader) (*http.Request, error)
}

type roxctlClientImpl struct {
	http        *http.Client
	a           Auth
	forceHTTP1  bool
	useInsecure bool
}

// GetRoxctlHTTPClient returns a new instance of RoxctlHTTPClient with the given configuration
func GetRoxctlHTTPClient(timeout time.Duration, forceHTTP1 bool, useInsecure bool) (RoxctlHTTPClient, error) {
	tlsConf, err := tlsConfigForCentral()
	if err != nil {
		return nil, errors.Wrap(err, "instantiating TLS configuration for central")
	}
	transport := &http.Transport{
		TLSClientConfig: tlsConf,
	}
	if forceHTTP1 {
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

	auth, err := NewAuth()
	if err != nil {
		return nil, err
	}
	return &roxctlClientImpl{http: client, a: auth, forceHTTP1: forceHTTP1, useInsecure: useInsecure}, nil
}

// DoReqAndVerifyStatusCode executes a http.Request and verifies that the http.Response had the given status code
func (client *roxctlClientImpl) DoReqAndVerifyStatusCode(path string, method string, code int, body io.Reader) (*http.Response, error) {
	req, err := client.NewReq(method, path, body)
	if err != nil {
		return nil, err
	}

	resp, err := client.http.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != code {
		defer utils.IgnoreError(resp.Body.Close)
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "Expected status code %d, but received %d. Additionally, there was an error reading the response", code, resp.StatusCode)
		}
		return nil, errors.Errorf("Expected status code %d, but received %d. Response Body: %s", code, resp.StatusCode, string(data))
	}

	return resp, nil
}

// NewReq creates a new http.Request which will have all authentication metadata injected
func (client *roxctlClientImpl) NewReq(method string, path string, body io.Reader) (*http.Request, error) {
	reqURL, err := getURL(path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, err
	}
	if client.forceHTTP1 {
		req.ProtoMajor, req.ProtoMinor, req.Proto = 1, 1, "HTTP/1.1"
	}

	if req.URL.Scheme != "https" && !client.useInsecure {
		return nil, errors.Errorf("URL %v uses insecure scheme %q, use --insecure flags to enable sending credentials", req.URL, req.URL.Scheme)
	}
	err = client.a.SetAuth(req)
	if err != nil {
		return nil, err
	}

	return req, nil
}
