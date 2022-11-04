package registry

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type LogfCallback func(format string, args ...interface{})

/*
 * Discard log messages silently.
 */
func Quiet(format string, args ...interface{}) {
	/* discard logs */
}

/*
 * Pass log messages along to Go's "log" module.
 */
func Log(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// Transport is the interface that all transports must implement so that a token can be retrieved from the registry
type Transport interface {
	http.RoundTripper
	GetToken() string
}

type Registry struct {
	URL       string
	Client    *http.Client
	Transport Transport
	Logf      LogfCallback
}

/*
 * Create a new Registry with the given URL and credentials, then Ping()s it
 * before returning it to verify that the registry is available.
 *
 * You can, alternately, construct a Registry manually by populating the fields.
 * This passes http.DefaultTransport to WrapTransport when creating the
 * http.Client.
 */
func New(registryUrl, username, password string) (*Registry, error) {
	transport := http.DefaultTransport

	return newWithWrapTransport(registryUrl, username, password, transport, Log)
}

/*
 * Create a new Registry, as with New, using an http.Transport that disables
 * SSL certificate verification.
 */
func NewInsecure(registryUrl, username, password string) (*Registry, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return newWithWrapTransport(registryUrl, username, password, transport, Log)
}

/*
 * Given an existing http.RoundTripper such as http.DefaultTransport, build the
 * transport stack necessary to authenticate to the Docker registry API. This
 * adds in support for OAuth bearer tokens and HTTP Basic auth, and sets up
 * error handling this library relies on.
 */
func WrapTransport(transport http.RoundTripper, url, username, password string) Transport {
	tokenTransport := &TokenTransport{
		Transport: transport,
		Username:  username,
		Password:  password,
	}
	basicAuthTransport := &BasicTransport{
		Transport: tokenTransport,
		URL:       url,
		Username:  username,
		Password:  password,
	}
	errorTransport := &ErrorTransport{
		Transport: basicAuthTransport,
	}
	return errorTransport
}

func newWithWrapTransport(registryUrl, username, password string, transport http.RoundTripper, logf LogfCallback) (*Registry, error) {
	url := strings.TrimSuffix(registryUrl, "/")
	wrappedTransport := WrapTransport(transport, url, username, password)
	return NewFromTransport(registryUrl, wrappedTransport, logf)
}

func NewFromTransport(registryUrl string, transport Transport, logf LogfCallback) (*Registry, error) {
	registry := &Registry{
		URL: registryUrl,
		Client: &http.Client{
			Transport: transport,
		},
		Transport: transport,
		Logf:      logf,
	}

	return registry, nil
}

func (r *Registry) url(pathTemplate string, args ...interface{}) string {
	pathSuffix := fmt.Sprintf(pathTemplate, args...)
	url := fmt.Sprintf("%s%s", r.URL, pathSuffix)
	return url
}

func (r *Registry) Ping() error {
	url := r.url("/v2/")
	r.Logf("registry.ping url=%s", url)
	resp, err := r.Client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// We read the results a little early so that, if the body exists,
	// we can print it out in the response for easier debuggability.
	results, readErr := ioutil.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorBuilder := strings.Builder{}
		errorBuilder.WriteString(fmt.Sprintf("got error status code %d pinging %s: %s",
			resp.StatusCode, url, resp.Status))
		if readErr != nil {
			errorBuilder.WriteString(fmt.Sprintf(" (body: %s)", results))
		}
		return NewClientError(resp.StatusCode, errors.New(errorBuilder.String()))
	}

	return readErr
}
