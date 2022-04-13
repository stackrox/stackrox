package endpoint

import (
	"crypto/tls"
	"net/http"

	"github.com/stackrox/stackrox/pkg/httputil/proxy"
)

var insecureHTTPClient = &http.Client{
	Transport: proxy.RoundTripperWithTLSConfig(&tls.Config{
		InsecureSkipVerify: true,
	}),
}
