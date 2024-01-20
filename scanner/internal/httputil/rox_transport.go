package httputil

import (
	"net"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/mtls"
)

// defaultDialer is copied from http.DefaultTransport as of go1.20.10.
var defaultDialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

// RoxTransportOptions represents transport options for reaching out to Rox-related services.
type RoxTransportOptions struct {
	disableCompression bool
}

// RoxTransport returns a [http.RoundTripper] capable of reaching out to a Rox-related service via mTLS.
func RoxTransport(subject mtls.Subject, o RoxTransportOptions) (http.RoundTripper, error) {
	tlsConfig, err := clientconn.TLSConfig(subject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})

	if err != nil {
		return nil, err
	}
	return &http.Transport{
		Proxy:              proxy.FromConfig(),
		TLSClientConfig:    tlsConfig,
		DisableCompression: o.disableCompression,

		// The rest are (more-or-less) copied from http.DefaultTransport as of go1.20.10.
		DialContext:           defaultDialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}, nil
}
