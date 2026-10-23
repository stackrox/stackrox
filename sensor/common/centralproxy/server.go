package centralproxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/mtls/certwatch"
	"github.com/stackrox/rox/pkg/mtls/clientca"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"k8s.io/client-go/kubernetes"
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
	proxyHandler, ok := h.(*Handler)
	managedPolicy := ok && proxyHandler.policy != nil
	if env.SecuredClusterMetricsEnabled.BooleanSetting() || managedPolicy {
		config, err := k8sutil.GetK8sInClusterConfig()
		if err != nil {
			return nil, errors.Wrap(err, "getting client CA Kubernetes config")
		}
		client, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, errors.Wrap(err, "creating client CA Kubernetes client")
		}
		tlsConfig = newMetricsProxyTLSConfig(cert, client)
		if managedPolicy {
			allowPolicyClientCertificates(tlsConfig)
		}
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

	go func() {
		log.Infof("Starting proxy server on %s with OpenShift service CA signed certificate", env.CentralProxyPort.Setting())
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Errorf("Proxy server failed: %v", err)
		}
	}()

	return server, nil
}

func allowPolicyClientCertificates(config *tls.Config) {
	// Certificate verification is performed against the matching policy entry's
	// CA in the handler, not against a common handshake trust pool.
	config.ClientAuth = tls.RequestClientCert
	getConfig := config.GetConfigForClient
	config.GetConfigForClient = func(info *tls.ClientHelloInfo) (*tls.Config, error) {
		live, err := getConfig(info)
		if err != nil {
			return nil, err
		}
		live.ClientAuth = tls.RequestClientCert
		live.ClientCAs = nil
		return live, nil
	}
}

// newMetricsProxyTLSConfig verifies presented client certificates against the
// OpenShift client CA without requiring certificates for console proxy callers.
func newMetricsProxyTLSConfig(initialCert tls.Certificate, client kubernetes.Interface) *tls.Config {
	config, updateClientCA, updateServerCert := proxyTLSConfig(initialCert)
	watcher := clientca.NewWatcher(client, updateClientCA)
	watcher.Watch(context.Background(), env.SecureMetricsClientCANamespace.Setting(),
		env.SecureMetricsClientCAConfigMap.Setting())

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
	return config
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
		live.ClientCAs = clientCAs.Load()
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
