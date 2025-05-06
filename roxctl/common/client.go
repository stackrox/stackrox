package common

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/auth"
	"golang.org/x/net/http2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	http1NextProtos = []string{"http/1.1", "http/1.0"}

	// RoxctlCommand is the reconstructed roxctl command line.
	RoxctlCommand string

	// RoxctlCommandIndex is the index of the current API call for the command.
	RoxctlCommandIndex atomic.Uint32
)

// RoxctlHTTPClient abstracts all HTTP-related functionalities required within roxctl
type RoxctlHTTPClient interface {
	DoReqAndVerifyStatusCode(path string, method string, code int, body io.Reader) (*http.Response, error)
	Do(req *http.Request) (*http.Response, error)
	NewReq(method string, path string, body io.Reader) (*http.Request, error)
}

type roxctlClientImpl struct {
	http        *http.Client
	am          auth.Method
	forceHTTP1  bool
	useInsecure bool
}

func getURL(path string) (string, error) {
	endpoint, _, usePlaintext, err := ConnectNames()
	if err != nil {
		return "", errors.Wrap(err, "could not get endpoint")
	}
	scheme := "https"
	if usePlaintext {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s/%s", scheme, endpoint, strings.TrimLeft(path, "/")), nil
}

// GetRoxctlHTTPClient returns a new instance of RoxctlHTTPClient with the given configuration
func GetRoxctlHTTPClient(config *HttpClientConfig) (RoxctlHTTPClient, error) {
	tlsConf, err := tlsConfigForCentral(config.Logger)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating TLS configuration for central")
	}
	transport := &http.Transport{
		TLSClientConfig: tlsConf,
	}
	if config.ForceHTTP1 {
		transport.TLSClientConfig.NextProtos = http1NextProtos
	} else {
		// There's no reason to not use HTTP/2, but we don't go out of our way to do so.
		if err := http2.ConfigureTransport(transport); err != nil {
			transport.TLSClientConfig.NextProtos = http1NextProtos
		}
	}

	retryClient := retryablehttp.NewClient()
	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		retry, err := retryablehttp.ErrorPropagatedRetryPolicy(ctx, resp, err)
		if !retry || status.Code(err) == codes.PermissionDenied {
			return false, err //nolint:wrapcheck
		}
		if err != nil {
			config.Logger.WarnfLn(err.Error())
		}
		return true, err //nolint:wrapcheck
	}
	retryClient.RetryMax = config.RetryCount
	retryClient.HTTPClient.Transport = transport
	retryClient.HTTPClient.Timeout = config.Timeout
	retryClient.RetryWaitMin = config.RetryDelay
	// Silence the default log output of the HTTP retry client to not pollute output.
	retryClient.Logger = nil

	if !config.RetryExponentialBackoff {
		// Disable the exponential backoff, in some scenarios the backoff makes roxctl appear
		// stuck (partially due to the logger being disabled).
		retryClient.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration { return min }
	}

	client := retryClient.StandardClient()
	return &roxctlClientImpl{http: client, am: config.AuthMethod, forceHTTP1: config.ForceHTTP1, useInsecure: config.UseInsecure}, nil
}

// DoReqAndVerifyStatusCode executes a http.Request and verifies that the http.Response had the given status code
func (client *roxctlClientImpl) DoReqAndVerifyStatusCode(path string, method string, code int, body io.Reader) (*http.Response, error) {
	req, err := client.NewReq(method, path, body)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != code {
		defer utils.IgnoreError(resp.Body.Close)
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "Expected status code %d, but received %d. Additionally, there was an error reading the response", code, resp.StatusCode)
		}
		return nil, errox.InvariantViolation.Newf("expected status code %d, but received %d. Response Body: %s", code, resp.StatusCode, string(data))
	}

	return resp, nil
}

func sanitizeHeaderValue(value string) string {
	return strings.Map(func(r rune) rune {
		// Allowed characters for a header value are all visible ASCII, which
		// are the runes in the range [33, 126].
		// They include field separators like brackets and such. See RFC7230.
		if r >= 33 && r <= 126 {
			return r
		}
		return ' '
	}, value)
}

func setCustomHeaders(headers func(string, ...string)) {
	headers(clientconn.RoxctlCommandHeader, RoxctlCommand)
	headers(clientconn.RoxctlCommandIndexHeader, fmt.Sprint(RoxctlCommandIndex.Add(1)))
	if e := env.ExecutionEnvironment.Setting(); e != "" {
		headers(clientconn.ExecutionEnvironment, sanitizeHeaderValue(e))
	}
}

// Do executes a http.Request
func (client *roxctlClientImpl) Do(req *http.Request) (*http.Response, error) {
	setCustomHeaders(phonehome.Headers(req.Header).Set)

	resp, err := client.http.Do(req)
	// The url.Error returned by go-retryablehttp needs to be unwrapped to retrieve the correct timeout settings.
	// See https://github.com/hashicorp/go-retryablehttp/issues/142.
	if _, ok := err.(*url.Error); ok {
		err = errors.Unwrap(err)
	}
	return resp, errors.Wrap(err, "error when doing http request")
}

// NewReq creates a new http.Request which will have all authentication metadata injected
func (client *roxctlClientImpl) NewReq(method string, path string, body io.Reader) (*http.Request, error) {
	reqURL, err := getURL(path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, errors.Wrap(err, "error when creating http request")
	}
	if client.forceHTTP1 {
		req.ProtoMajor, req.ProtoMinor, req.Proto = 1, 1, "HTTP/1.1"
	}

	creds, err := client.am.GetCredentials(req.URL.Hostname() + ":" + req.URL.Port())
	if err != nil {
		return nil, errors.Wrapf(err, "could not obtain credentials for %s", reqURL)
	}

	if creds.RequireTransportSecurity() && req.URL.Scheme != "https" && !client.useInsecure {
		return nil, errox.InvalidArgs.Newf("URL %v uses insecure scheme %q, use --insecure flags to enable sending credentials", req.URL, req.URL.Scheme)
	}

	// Add all headers containing authentication information to the request.
	md, err := creds.GetRequestMetadata(req.Context(), reqURL)
	if err != nil {
		return nil, errors.Wrap(err, "could not inject authentication information")
	}
	for k, v := range md {
		req.Header.Add(k, v)
	}

	req.Header.Set("User-Agent", clientconn.GetUserAgent())

	return req, nil
}
