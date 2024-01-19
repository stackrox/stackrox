package httputil

import (
	"net/http"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/mtls"
)

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
	}, nil
}
