package httputil

import (
	"net/http"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/mtls"
)

// RoxClientOptions represents client options for reaching out to Rox-services.
type RoxClientOptions struct {
	disableCompression bool
}

func RoxTransport(subject mtls.Subject, o RoxClientOptions) (http.RoundTripper, error) {
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
