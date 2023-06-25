package metrics

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"path/filepath"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/k8scfgmap"
	"github.com/stackrox/rox/pkg/mtls/certwatch"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/sync"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	k8sWatcher        *k8scfgmap.ConfigMapWatcher

	clientCAs       []*x509.Certificate
	serverCerts     []tls.Certificate
	tlsConfigHolder *certwatch.TLSConfigHolder

	mutex sync.RWMutex
}

var _ TLSConfigurer = (*tlsConfigurerImpl)(nil)

// newTLSConfigurer creates a new TLS configurer.
func newTLSConfigurer(certDir string, k8sClient kubernetes.Interface, clientCANamespace, clientCAConfigMap string) TLSConfigurer {
	tlsRootConfig := verifier.DefaultTLSServerConfig(nil, nil)
	tlsRootConfig.ClientAuth = tls.RequireAndVerifyClientCert
	cfgr := &tlsConfigurerImpl{
		certDir:           certDir,
		clientCANamespace: clientCANamespace,
		clientCAConfigMap: clientCAConfigMap,
		tlsConfigHolder:   certwatch.NewTLSConfigHolder(tlsRootConfig),
	}
	cfgr.tlsConfigHolder.AddServerCertSource(&cfgr.serverCerts)
	cfgr.tlsConfigHolder.AddClientCertSource(&cfgr.clientCAs)
	cfgr.k8sWatcher = k8scfgmap.NewConfigMapWatcher(k8sClient, cfgr.updateClientCA)
	return cfgr
}

// NewTLSConfigurerFromEnv creates a new TLS configurer based on environment variables.
func NewTLSConfigurerFromEnv() TLSConfigurer {
	if !secureMetricsEnabled() {
		return &nilTLSConfigurer{}
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil
	}
	certDir := env.SecureMetricsCertDir.Setting()
	clientCANamespace := env.SecureMetricsClientCANamespace.Setting()
	clientCAConfigMap := env.SecureMetricsClientCAConfigMap.Setting()
	cfgr := newTLSConfigurer(certDir, clientset, clientCANamespace, clientCAConfigMap)
	return cfgr
}

// WatchForChanges watches for changes of the server TLS certificate files and the client CA config map.
func (t *tlsConfigurerImpl) WatchForChanges() {
	// Watch for changes of server TLS certificate.
	certwatch.WatchCertDir(t.certDir, t.getCertificateFromDirectory, t.updateCertificate)

	// Watch for changes of client CA.
	go t.k8sWatcher.Watch(context.Background(), t.clientCANamespace, t.clientCAConfigMap)
}

// TLSConfig returns the current TLS config.
func (t *tlsConfigurerImpl) TLSConfig() (*tls.Config, error) {
	if t == nil {
		return nil, nil
	}
	return t.tlsConfigHolder.TLSConfig()
}

func (t *tlsConfigurerImpl) getCertificateFromDirectory(dir string) (*tls.Certificate, error) {
	certFile := filepath.Join(dir, env.TLSCertFileName)
	if exists, err := fileutils.Exists(certFile); err != nil || !exists {
		if err != nil {
			log.Warnw("Error checking if monitoring TLS certificate file exists", zap.Error(err))
			return nil, err
		}
		log.Infof("Monitoring TLS certificate file %q does not exist. Skipping", certFile)
		return nil, nil
	}

	keyFile := filepath.Join(dir, env.TLSKeyFileName)
	if exists, err := fileutils.Exists(keyFile); err != nil || !exists {
		if err != nil {
			log.Warnw("Error checking if monitoring TLS key file exists", zap.Error(err))
			return nil, err
		}
		log.Infof("Monitoring TLS key file %q does not exist. Skipping", keyFile)
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Wrap(err, "loading monitoring certificate failed")
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, errors.Wrap(err, "parsing leaf certificate failed")
	}
	return &cert, nil
}

func (t *tlsConfigurerImpl) updateCertificate(cert *tls.Certificate) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if cert == nil {
		t.serverCerts = nil
	} else {
		t.serverCerts = []tls.Certificate{*cert}
	}
	t.tlsConfigHolder.UpdateTLSConfig()
}

func (t *tlsConfigurerImpl) updateClientCA(cm *v1.ConfigMap) {
	if cm == nil {
		return
	}
	if caFile, ok := cm.Data["client-ca-file"]; ok {
		certPEM := []byte(caFile)
		certBlock, _ := pem.Decode([]byte(certPEM))
		cert, err := x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			log.Errorw("Unable to parse client CA", zap.Error(err))
			return
		}
		t.mutex.Lock()
		t.clientCAs = []*x509.Certificate{cert}
		t.tlsConfigHolder.UpdateTLSConfig()
		t.mutex.Unlock()
	}
}
