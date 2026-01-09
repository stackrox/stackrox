package centralproxy

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const (
	proxyCertPath = "/run/secrets/stackrox.io/proxy-tls/tls.crt"
	proxyKeyPath  = "/run/secrets/stackrox.io/proxy-tls/tls.key"
	proxyPort     = ":9444"
)

// StartProxyServer starts a dedicated HTTP server for the /proxy/central endpoint
// using an OpenShift service CA signed certificate. Returns the server instance for
// lifecycle management (e.g., graceful shutdown).
func StartProxyServer(h http.Handler) (*http.Server, error) {
	cert, err := tls.LoadX509KeyPair(proxyCertPath, proxyKeyPath)
	if err != nil {
		return nil, errors.Wrap(err, "loading proxy TLS certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	mux := http.NewServeMux()
	mux.Handle("/proxy/central/", http.StripPrefix("/proxy/central", h))

	server := &http.Server{
		Addr:              proxyPort,
		Handler:           mux,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Infof("Starting proxy server on %s with OpenShift service CA signed certificate", proxyPort)
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Errorf("Proxy server failed: %v", err)
		}
	}()

	return server, nil
}
