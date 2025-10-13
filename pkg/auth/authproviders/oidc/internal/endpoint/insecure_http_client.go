package endpoint

import (
	"crypto/tls"
	"net/http"

	"github.com/stackrox/rox/pkg/httputil/proxy"
)

var insecureHTTPClient = &http.Client{
	Transport: proxy.RoundTripper(
		proxy.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}),
	),
}
