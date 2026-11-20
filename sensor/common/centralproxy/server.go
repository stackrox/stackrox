package centralproxy

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/mtls/certwatch"
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

	tlsConfig, updateServerCert := proxyTLSConfig(cert)
	watchProxyServingCertificate(updateServerCert)
	proxyHandler, ok := h.(*Handler)
	if ok && proxyHandler.policy != nil {
		proxyHandler.policy.configureTLS(tlsConfig)
	}

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
	if ok && proxyHandler.policy != nil {
		server.RegisterOnShutdown(proxyHandler.policy.Stop)
	}

	go func() {
		log.Infof("Starting proxy server on %s with OpenShift service CA signed certificate", env.CentralProxyPort.Setting())
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Errorf("Proxy server failed: %v", err)
		}
	}()

	return server, nil
}

func watchProxyServingCertificate(updateServerCert func(*tls.Certificate)) {
	certwatch.WatchCertDir("sensor proxy", filepath.Dir(env.CentralProxyCertPath.Setting()),
		func(string) (*tls.Certificate, error) {
			cert, err := tls.LoadX509KeyPair(env.CentralProxyCertPath.Setting(), env.CentralProxyKeyPath.Setting())
			if err != nil {
				return nil, errors.Wrap(err, "loading rotated proxy certificate")
			}
			cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
			if err != nil {
				return nil, errors.Wrap(err, "parsing rotated proxy certificate")
			}
			return &cert, nil
		}, updateServerCert, certwatch.WithVerify(false))
}

func proxyTLSConfig(initialCert tls.Certificate) (*tls.Config, func(*tls.Certificate)) {
	var serverCert atomic.Pointer[tls.Certificate]
	serverCert.Store(&initialCert)

	config := verifier.DefaultTLSServerConfig(nil, []tls.Certificate{initialCert})
	// Disable resumption so each connection negotiates its client certificate.
	config.SessionTicketsDisabled = true
	config.GetConfigForClient = func(*tls.ClientHelloInfo) (*tls.Config, error) {
		live := config.Clone()
		live.GetConfigForClient = nil
		live.Certificates = []tls.Certificate{*serverCert.Load()}
		return live, nil
	}

	updateServerCert := func(cert *tls.Certificate) {
		if cert != nil {
			serverCert.Store(cert)
		}
	}
	return config, updateServerCert
}
