package k8swatch

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	caCertPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

var (
	inClusterClient     *http.Client
	inClusterClientOnce sync.Once
)

// InClusterClient returns an HTTP client configured for in-cluster k8s API access.
func InClusterClient() *http.Client {
	inClusterClientOnce.Do(func() {
		tlsConfig := &tls.Config{}

		if caCert, err := os.ReadFile(caCertPath); err == nil {
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(caCert)
			tlsConfig.RootCAs = pool
		}

		inClusterClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:     tlsConfig,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	})
	return inClusterClient
}
