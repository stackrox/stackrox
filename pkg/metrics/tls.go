package metrics

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"path/filepath"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/mtls/certwatch"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/sync"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	clientCAKey = "client-ca-file"
	signerName  = "kubelet-signer"
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

// nilTLSConfigurer is a no-op configurer.
type nilTLSConfigurer struct{}

// TLSConfig returns nil.
func (t *nilTLSConfigurer) TLSConfig() (*tls.Config, error) {
	return nil, nil
}

var _ verifier.TLSConfigurer = (*nilTLSConfigurer)(nil)

// tlsConfigurerImpl holds the current TLS configuration.
//
// The TLS configuration contains both server certificates and client CA, which
// are both watched for changes to dynamically reload the TLS configuration.
// The server certificates are read from file-mounted secrets. The client CA is
// read from an external config map via the Kubernetes API.
type tlsConfigurerImpl struct {
	certDir           string
	clientCAConfigMap string
	clientCANamespace string
	k8sWatcher        *k8scfgwatch.ConfigMapWatcher

	clientCAs       []*x509.Certificate
	serverCerts     []tls.Certificate
	tlsConfigHolder *certwatch.TLSConfigHolder

	mutex sync.Mutex
}

var _ verifier.TLSConfigurer = (*tlsConfigurerImpl)(nil)

// newTLSConfigurer creates a new TLS configurer.
func newTLSConfigurer(certDir string, k8sClient kubernetes.Interface, clientCANamespace, clientCAConfigMap string) verifier.TLSConfigurer {
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
	cfgr.k8sWatcher = k8scfgwatch.NewConfigMapWatcher(k8sClient, cfgr.updateClientCA)
	cfgr.watchForChanges()
	return cfgr
}

// NewTLSConfigurerFromEnv creates a new TLS configurer based on environment variables.
func NewTLSConfigurerFromEnv() verifier.TLSConfigurer {
	if !secureMetricsEnabled() {
		return &nilTLSConfigurer{}
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Errorw("Failed to get in-cluster config", zap.Error(err))
		return &nilTLSConfigurer{}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorw("Failed to create Kubernetes client", zap.Error(err))
		return &nilTLSConfigurer{}
	}
	certDir := env.SecureMetricsCertDir.Setting()
	clientCANamespace := env.SecureMetricsClientCANamespace.Setting()
	clientCAConfigMap := env.SecureMetricsClientCAConfigMap.Setting()
	cfgr := newTLSConfigurer(certDir, clientset, clientCANamespace, clientCAConfigMap)
	return cfgr
}

// watchForChanges watches for changes of the server TLS certificate files and the client CA config map.
func (t *tlsConfigurerImpl) watchForChanges() {
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
			log.Errorw("Error checking if monitoring TLS certificate file exists", zap.Error(err))
			return nil, err
		}
		log.Infof("Monitoring TLS certificate file %q does not exist. Skipping TLS watcher cycle.", certFile)
		return nil, nil
	}

	keyFile := filepath.Join(dir, env.TLSKeyFileName)
	if exists, err := fileutils.Exists(keyFile); err != nil || !exists {
		if err != nil {
			log.Errorw("Error checking if monitoring TLS key file exists", zap.Error(err))
			return nil, err
		}
		log.Infof("Monitoring TLS key file %q does not exist. Skipping TLS watcher cycle.", keyFile)
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
	if caFile, ok := cm.Data[clientCAKey]; ok {
		log.Infof("Updating secure metrics client CAs based on %s/%s", t.clientCANamespace, t.clientCAConfigMap)
		certs, err := helpers.ParseCertificatesPEM([]byte(caFile))
		if err != nil {
			log.Errorw("Unable to parse client CAs", zap.Error(err))
			return
		}
		var signerCAs []*x509.Certificate
		for _, c := range certs {
			if c.Issuer.CommonName == signerName {
				signerCAs = append(signerCAs, c)
			}
		}
		t.mutex.Lock()
		defer t.mutex.Unlock()
		t.clientCAs = signerCAs
		t.tlsConfigHolder.UpdateTLSConfig()
	}
}
