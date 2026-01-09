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

// newCentralReverseProxy creates a new ReverseProxy that forwards requests to Central.
// Uses httputil.ReverseProxy which automatically handles hop-by-hop headers per RFC 7230.
func newCentralReverseProxy(baseURL *url.URL, certs []*x509.Certificate, token string) (*httputil.ReverseProxy, error) {
	certPool, err := verifier.SystemCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "getting system cert pool")
	}
	for _, cert := range certs {
		certPool.AddCert(cert)
	}

	baseTransport := pkghttputil.DefaultTransport()
	baseTransport.TLSClientConfig = &tls.Config{
		RootCAs:    certPool,
		ServerName: baseURL.Hostname(),
	}

	// Minimal wrapper that adds Bearer token authentication.
	tokenTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		return baseTransport.RoundTrip(req)
	})

	return &httputil.ReverseProxy{
		Transport: tokenTransport,
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(baseURL)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			pkghttputil.WriteGRPCStyleErrorf(w, codes.Internal, "failed to contact central: %v", err)
		},
	}, nil
}
