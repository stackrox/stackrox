package tlsconfig

import (
	"context"
	"crypto/tls"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls/certwatch"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// APIMonitoringConfigured reports whether the OpenShift API monitoring certificate mount is present.
func APIMonitoringConfigured() bool {
	exists, err := fileutils.Exists(env.APIMonitoringCertDir.Setting())
	return exists && err == nil
}

func (m *managerImpl) initAPIMonitoringTLS() {
	if !APIMonitoringConfigured() {
		return
	}
	certwatch.WatchCertDir(
		"API monitoring TLS",
		env.APIMonitoringCertDir.Setting(),
		MaybeGetDefaultTLSCertificateFromDirectory,
		m.UpdateMonitoringTLSCertificate,
		certwatch.WithVerify(false),
	)

	config, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		log.Errorw("Failed to get in-cluster config for API monitoring client CA watch", logging.Err(err))
		return
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorw("Failed to create Kubernetes client for API monitoring client CA watch", logging.Err(err))
		return
	}

	watcher := k8scfgwatch.NewConfigMapWatcher(clientset, m.updateMonitoringClientCA)
	watcher.Watch(
		context.Background(),
		env.SecureMetricsClientCANamespace.Setting(),
		env.SecureMetricsClientCAConfigMap.Setting(),
	)
}

func (m *managerImpl) UpdateMonitoringTLSCertificate(cert *tls.Certificate) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if cert == nil {
		m.monitoringCerts = nil
	} else {
		m.monitoringCerts = []tls.Certificate{*cert}
	}
	m.updateConfigurersNoLock()
}

func (m *managerImpl) updateMonitoringClientCA(cm *v1.ConfigMap) {
	if cm == nil {
		return
	}
	caFile, ok := cm.Data[env.SecureMetricsClientCAKey.Setting()]
	if !ok {
		return
	}

	log.Debugf("Updating API monitoring client CAs based on %s/%s",
		env.SecureMetricsClientCANamespace.Setting(),
		env.SecureMetricsClientCAConfigMap.Setting(),
	)
	signerCAs, err := helpers.ParseCertificatesPEM([]byte(caFile))
	if err != nil {
		log.Errorw("Unable to parse API monitoring client CAs", logging.Err(err))
		return
	}
	if len(signerCAs) == 0 {
		log.Warnf("No API monitoring client CAs have been found in %q/%q",
			env.SecureMetricsClientCANamespace.Setting(),
			env.SecureMetricsClientCAConfigMap.Setting(),
		)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.monitoringClientCAs = signerCAs
	m.updateConfigurersNoLock()
}
