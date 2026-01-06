package centralproxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/pkg/errors"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"google.golang.org/grpc/codes"
)

// tokenTransport is a custom RoundTripper that adds Bearer token authentication.
type tokenTransport struct {
	base  http.RoundTripper
	token string
}

// RoundTrip adds the Authorization header with Bearer token to the request.
func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	modReq := req.Clone(req.Context())
	modReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.token))
	return t.base.RoundTrip(modReq) //nolint:wrapcheck
}

// newHTTPTransportWithToken creates an HTTP transport with TLS config and token authentication.
func newHTTPTransportWithToken(baseURL *url.URL, certs []*x509.Certificate, token string) (http.RoundTripper, error) {
	certPool, err := verifier.SystemCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "getting system cert pool")
	}
	for _, cert := range certs {
		certPool.AddCert(cert)
	}

	transport := pkghttputil.DefaultTransport()
	transport.TLSClientConfig = &tls.Config{
		RootCAs:    certPool,
		ServerName: baseURL.Hostname(),
	}

	return &tokenTransport{
		base:  transport,
		token: token,
	}, nil
}

// newProxy creates a new ReverseProxy that forwards requests to Central.
// Uses httputil.ReverseProxy which automatically handles hop-by-hop headers per RFC 7230.
func newProxy(baseURL *url.URL, transport http.RoundTripper) *httputil.ReverseProxy {
	// Create ReverseProxy with Rewrite for cleaner URL handling.
	// ReverseProxy automatically handles hop-by-hop headers per RFC 7230.
	return &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(r *httputil.ProxyRequest) {
			// SetURL automatically sets the target URL's Scheme, Host, and Path.
			// It also sets r.Out.Host to the target host, avoiding routing/CSRF issues.
			r.SetURL(baseURL)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			pkghttputil.WriteGRPCStyleErrorf(w, codes.Internal, "failed to contact central: %v", err)
		},
	}
}
