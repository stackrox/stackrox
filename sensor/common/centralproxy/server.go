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

	tlsConfig, updateClientCA, updateServerCert := proxyTLSConfig(cert)
	watchProxyServingCertificate(updateServerCert)
	proxyHandler, ok := h.(*Handler)
	if ok && proxyHandler.policy != nil {
		proxyHandler.policy.configureTLS(tlsConfig, updateClientCA)
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
	if ok {
		if policy, stoppable := proxyHandler.policy.(interface{ Stop() }); stoppable {
			server.RegisterOnShutdown(policy.Stop)
		}
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

func proxyTLSConfig(initialCert tls.Certificate) (*tls.Config, func([]*x509.Certificate), func(*tls.Certificate)) {
	var clientCAs atomic.Pointer[x509.CertPool]
	clientCAs.Store(x509.NewCertPool())
	var serverCert atomic.Pointer[tls.Certificate]
	serverCert.Store(&initialCert)

	config := verifier.DefaultTLSServerConfig(nil, []tls.Certificate{initialCert})
	// A resumed session must not bypass client certificate verification after
	// OpenShift rotates its trusted client CA.
	config.SessionTicketsDisabled = true
	config.GetConfigForClient = func(*tls.ClientHelloInfo) (*tls.Config, error) {
		live := config.Clone()
		live.GetConfigForClient = nil
		live.Certificates = []tls.Certificate{*serverCert.Load()}
		if live.ClientAuth == tls.VerifyClientCertIfGiven {
			live.ClientCAs = clientCAs.Load()
		}
		return live, nil
	}

	updateClientCA := func(certs []*x509.Certificate) {
		pool := x509.NewCertPool()
		if len(certs) == 0 {
			log.Warn("OpenShift Prometheus client CA is missing or invalid; retaining previous CA")
			return
		}
		for _, cert := range certs {
			pool.AddCert(cert)
		}
		clientCAs.Store(pool)
	}
	updateServerCert := func(cert *tls.Certificate) {
		if cert != nil {
			serverCert.Store(cert)
		}
	}
	return config, updateClientCA, updateServerCert
}
