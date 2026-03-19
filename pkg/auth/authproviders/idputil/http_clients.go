package idputil

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/httputil/proxy"
)

const (
	// defaultTimeout is the timeout for HTTP requests to external IdPs.
	// This prevents hanging calls while still allowing enough time for typical
	// authentication flows.
	defaultTimeout = 30 * time.Second
)

// NewHTTPClient creates a proxy-aware HTTP client for secure IdP connections.
// The client includes a timeout to prevent hanging calls to external IdPs.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Transport: proxy.RoundTripper(),
		Timeout:   defaultTimeout,
	}
}

// NewInsecureHTTPClient creates a proxy-aware HTTP client that skips TLS verification.
// This should only be used when the IdP URL contains the "+insecure" scheme suffix.
// The client includes a timeout to prevent hanging calls to external IdPs.
func NewInsecureHTTPClient() *http.Client {
	return &http.Client{
		Transport: proxy.RoundTripper(
			proxy.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}),
		),
		Timeout: defaultTimeout,
	}
}
