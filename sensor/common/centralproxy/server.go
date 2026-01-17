package centralproxy

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/mtls/verifier"
)

// StartProxyServer starts a dedicated HTTP server for the /proxy/central endpoint
// using an OpenShift service CA signed certificate. Returns the server instance for
// lifecycle management (e.g., graceful shutdown).
func StartProxyServer(h http.Handler) (*http.Server, error) {
	cert, err := tls.LoadX509KeyPair(env.CentralProxyCertPath.Setting(), env.CentralProxyKeyPath.Setting())
	if err != nil {
		return nil, errors.Wrap(err, "loading proxy TLS certificate")
	}

	tlsConfig := verifier.DefaultTLSServerConfig(nil, []tls.Certificate{cert})

	mux := http.NewServeMux()
	mux.Handle("/proxy/central/", http.StripPrefix("/proxy/central", h))

	server := &http.Server{
		Addr:              env.CentralProxyPort.Setting(),
		Handler:           mux,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Infof("Starting proxy server on %s with OpenShift service CA signed certificate", env.CentralProxyPort.Setting())
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Errorf("Proxy server failed: %v", err)
		}
	}()

	return server, nil
}
