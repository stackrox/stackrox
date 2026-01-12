package centralproxy

import (
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
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

	// Use TLSConfig without client certificate to keep authn/z based on the user's
	// Bearer token, but include serviceCertFallbackVerifier for proper StackRox service
	// cert handling.
	tlsConf, err := clientconn.TLSConfig(mtls.CentralSubject, clientconn.TLSConfigOptions{
		ServerName:    baseURL.Hostname(),
		RootCAs:       certPool,
		UseClientCert: clientconn.DontUseClientCert,
	})
	if err != nil {
		return nil, errors.Wrap(err, "creating TLS config")
	}

	baseTransport := pkghttputil.DefaultTransport()
	baseTransport.TLSClientConfig = tlsConf

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
			pkghttputil.WriteError(w,
				pkghttputil.Errorf(http.StatusInternalServerError, "failed to contact central: %v", err),
			)
		},
	}, nil
}
