package tlsutils

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
)

const (
	// ServiceOperatorCAPath points to the service account secret which within
	// an OpenShift environment also contains the service-ca.crt. This CA is
	// used to verify certificates issued by the service-ca operator.
	ServiceOperatorCAPath = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
)

// TransportWithServiceCA returns an http.Transport whose TLS config trusts the
// system cert pool plus the OpenShift service-serving CA (if available).
func TransportWithServiceCA() *http.Transport {
	return TransportWithAdditionalCA(ServiceOperatorCAPath)
}

// TransportWithAdditionalCA returns an http.Transport whose TLS config trusts
// the system cert pool plus an additional CA file (if readable).
func TransportWithAdditionalCA(caFile string) *http.Transport {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = x509.NewCertPool()
	}

	if caData, err := os.ReadFile(caFile); err == nil {
		rootCAs.AppendCertsFromPEM(caData)
	}

	return &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    rootCAs,
		},
	}
}
