package tlsconfig

import (
	"crypto/tls"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/mtls/certwatch"
)

// OpenShiftTLSConfigured reports whether OpenShift service-serving TLS for the
// central-ocp Service should be enabled: Central must run on OpenShift and the
// certificate mount directory must be present.
func OpenShiftTLSConfigured() bool {
	if !env.Openshift.BooleanSetting() {
		return false
	}
	exists, err := fileutils.Exists(env.OpenShiftTLSCertDir.Setting())
	return exists && err == nil
}

func (m *managerImpl) initOpenShiftTLS() {
	if !OpenShiftTLSConfigured() {
		return
	}
	certwatch.WatchCertDir(
		"OpenShift service-serving TLS",
		env.OpenShiftTLSCertDir.Setting(),
		MaybeLoadOpenShiftTLSCertificateFromDirectory,
		m.UpdateOpenShiftTLSCertificate,
		certwatch.WithVerify(false),
	)
}

func (m *managerImpl) UpdateOpenShiftTLSCertificate(cert *tls.Certificate) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if cert == nil {
		m.openShiftCerts = nil
	} else {
		m.openShiftCerts = []tls.Certificate{*cert}
	}
	m.updateConfigurersNoLock()
}
