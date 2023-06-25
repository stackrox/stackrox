package metrics

import (
	"crypto/tls"
	"path/filepath"

	"github.com/pkg/errors"
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
//go:generate mockgen-wrapper
type TLSConfigurer interface {
	TLSConfig() (*tls.Config, error)
	WatchForChanges()
}

// NilTLSConfigurer is a no-op configurer.
type NilTLSConfigurer struct{}

// WatchForChanges does nothing.
func (t *NilTLSConfigurer) WatchForChanges() {}

// TLSConfig returns nil.
func (t *NilTLSConfigurer) TLSConfig() (*tls.Config, error) {
	return nil, nil
}

var _ TLSConfigurer = (*NilTLSConfigurer)(nil)

// TLSConfigurerImpl holds the current TLS configuration. The configurer
// watches the certificate directory for changes and updates the server
// certificates in the TLS config. The client CA is updated based on a
// Kubernetes config map watcher.
type TLSConfigurerImpl struct {
	certDir           string
	clientCAConfigMap string
	clientCANamespace string
}

var _ TLSConfigurer = (*TLSConfigurerImpl)(nil)

// NewTLSConfigurer creates a new TLS configurer.
func NewTLSConfigurer(certDir string, clientCANamespace, clientCAConfigMap string) (TLSConfigurer, error) {
	cfgr := &TLSConfigurerImpl{
		certDir:           certDir,
		clientCANamespace: clientCANamespace,
		clientCAConfigMap: clientCAConfigMap,
	}
	return cfgr, nil
}

// NewTLSConfigurerFromEnv creates a new TLS configurer based on environment variables.
func NewTLSConfigurerFromEnv() TLSConfigurer {
	if !secureMetricsEnabled() {
		return nil
	}

	certDir := env.SecureMetricsCertDir.Setting()
	clientCANamespace := env.SecureMetricsClientCANamespace.Setting()
	clientCAConfigMap := env.SecureMetricsClientCAConfigMap.Setting()
	cfgr, err := NewTLSConfigurer(certDir, clientCANamespace, clientCAConfigMap)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to create TLS config loader"))
	}
	return cfgr
}

// WatchForChanges watches for changes of the server TLS certificate files and the client CA config map.
func (t *TLSConfigurerImpl) WatchForChanges() {
}

// TLSConfig returns the current TLS config.
func (t *TLSConfigurerImpl) TLSConfig() (*tls.Config, error) {
	if t == nil {
		return nil, nil
	}
	return nil, nil
}
