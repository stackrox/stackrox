// Package clientca watches the OpenShift client CA used to verify Prometheus certificates.
package clientca

import (
	"crypto/x509"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/logging"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

var log = logging.LoggerForModule()

// OnUpdate reads the configured CA key and passes parsed certificates to the callback.
// Missing or malformed data leaves the previous CA unchanged.
func OnUpdate(update func([]*x509.Certificate)) func(*v1.ConfigMap) {
	return func(cm *v1.ConfigMap) {
		if cm == nil {
			return
		}
		pem, ok := cm.Data[env.SecureMetricsClientCAKey.Setting()]
		if !ok {
			return
		}
		certs, err := helpers.ParseCertificatesPEM([]byte(pem))
		if err != nil {
			log.Errorf("Unable to parse client CAs: %v", err)
			return
		}
		update(certs)
	}
}

// NewWatcher creates a ConfigMap watcher for the configured client CA key.
func NewWatcher(client kubernetes.Interface, update func([]*x509.Certificate)) *k8scfgwatch.ConfigMapWatcher {
	return k8scfgwatch.NewConfigMapWatcher(client, OnUpdate(update))
}
