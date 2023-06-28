package metrics

import (
	"crypto/tls"
	"path/filepath"

	"github.com/stackrox/rox/pkg/env"
)

func certFilePath() string {
	certDir := env.SecureMetricsCertDir.Setting()
	certFile := filepath.Join(certDir, env.TLSCertFileName)
	return certFile
}

func keyFilePath() string {
	certDir := env.SecureMetricsCertDir.Setting()
	keyFile := filepath.Join(certDir, env.TLSKeyFileName)
	return keyFile
}

// TLSConfigurer instantiates and updates the TLS configuration of a web server.
//
// The TLS configuration contains both server certificates and client CA, which
// are both watched for changes to dynamically reload the TLS configuration.
// The server certificates are read from file-mounted secrets. The client CA is
// read from an external config map via the Kubernetes API.
//
//go:generate mockgen-wrapper
type TLSConfigurer interface {
	TLSConfig() (*tls.Config, error)
	WatchForChanges()
}

// nilTLSConfigurer is a no-op configurer.
type nilTLSConfigurer struct{}

// WatchForChanges does nothing.
func (t *nilTLSConfigurer) WatchForChanges() {}

// TLSConfig returns nil.
func (t *nilTLSConfigurer) TLSConfig() (*tls.Config, error) {
	return nil, nil
}

var _ TLSConfigurer = (*nilTLSConfigurer)(nil)

// tlsConfigurerImpl holds the current TLS configuration. The configurer
// watches the certificate directory for changes and updates the server
// certificates in the TLS config. The client CA is updated based on a
// Kubernetes config map watcher.
type tlsConfigurerImpl struct {
	certDir           string
	clientCAConfigMap string
	clientCANamespace string
}

var _ TLSConfigurer = (*tlsConfigurerImpl)(nil)

// NewTLSConfigurer creates a new TLS configurer.
func NewTLSConfigurer(certDir string, clientCANamespace, clientCAConfigMap string) TLSConfigurer {
	cfgr := &tlsConfigurerImpl{
		certDir:           certDir,
		clientCANamespace: clientCANamespace,
		clientCAConfigMap: clientCAConfigMap,
	}
	return cfgr
}

// NewTLSConfigurerFromEnv creates a new TLS configurer based on environment variables.
func NewTLSConfigurerFromEnv() TLSConfigurer {
	if !secureMetricsEnabled() {
		return &nilTLSConfigurer{}
	}

	certDir := env.SecureMetricsCertDir.Setting()
	clientCANamespace := env.SecureMetricsClientCANamespace.Setting()
	clientCAConfigMap := env.SecureMetricsClientCAConfigMap.Setting()
	cfgr := NewTLSConfigurer(certDir, clientCANamespace, clientCAConfigMap)
	return cfgr
}

// WatchForChanges watches for changes of the server TLS certificate files and the client CA config map.
func (t *tlsConfigurerImpl) WatchForChanges() {
}

// TLSConfig returns the current TLS config.
func (t *tlsConfigurerImpl) TLSConfig() (*tls.Config, error) {
	if t == nil {
		return nil, nil
	}
	return nil, nil
}
