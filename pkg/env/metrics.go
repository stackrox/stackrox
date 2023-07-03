package env

import (
	"net"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fileutils"
)

const (
	// TLSCertFileName is the tls certificate filename.
	TLSCertFileName = "tls.crt"
	// TLSKeyFileName is the private key filename.
	TLSKeyFileName = "tls.key"

	defaultHTTPPort  = ":9090"
	defaultHTTPSPort = ":9091"
)

var (
	// MetricsPort has the :port or host:port string for listening for metrics/debug server.
	MetricsPort = RegisterSetting("ROX_METRICS_PORT", WithDefault(defaultHTTPPort))
	// EnableSecureMetrics enables the secure metrics endpoint.
	EnableSecureMetrics = RegisterBooleanSetting("ROX_ENABLE_SECURE_METRICS", false)
	// SecureMetricsPort has the :port or host:port string for listening for metrics/debug server.
	SecureMetricsPort = RegisterSetting("ROX_SECURE_METRICS_PORT", WithDefault(defaultHTTPSPort))
	// SecureMetricsCertDir has the server's TLS certificate files.
	SecureMetricsCertDir = RegisterSetting("ROX_SECURE_METRICS_CERT_DIR", WithDefault("/run/secrets/stackrox.io/monitoring-tls"))
	// SecureMetricsClientCANamespace has the namespace that contains the client CA.
	SecureMetricsClientCANamespace = RegisterSetting("ROX_SECURE_METRICS_CLIENT_CA_NS", WithDefault("kube-system"))
	// SecureMetricsClientCAConfigMap has the config map that contains the client CA.
	SecureMetricsClientCAConfigMap = RegisterSetting("ROX_SECURE_METRICS_CLIENT_CA_CFG", WithDefault("extension-apiserver-authentication"))
)

func validatePort(setting Setting) error {
	val := setting.Setting()
	addr, err := net.ResolveTCPAddr("tcp", val)
	if err != nil {
		return err
	}
	log.Debugf("%s=%s, resolved to %+v", setting.EnvVar(), val, addr)
	return nil
}

func validateTLS() error {
	certFile := filepath.Join(SecureMetricsCertDir.Setting(), TLSCertFileName)
	if ok, err := fileutils.Exists(certFile); !ok {
		if err != nil {
			log.Errorf("failed to validate file %q: %s", certFile, err.Error())
		}
		return errors.Wrapf(errox.NotFound, "secure metrics certificate file %q not found", certFile)
	}

	keyFile := filepath.Join(SecureMetricsCertDir.Setting(), TLSKeyFileName)
	if ok, err := fileutils.Exists(keyFile); !ok {
		if err != nil {
			log.Errorf("failed to validate file %q: %s", keyFile, err.Error())
		}
		return errors.Wrapf(errox.NotFound, "secure metrics key file %q not found", keyFile)
	}
	return nil
}

// ValidateMetricsSetting returns an error if the environment variable is invalid.
func ValidateMetricsSetting() error {
	if !MetricsEnabled() {
		return nil
	}
	return validatePort(MetricsPort)
}

// MetricsEnabled returns true if the metrics/debug http server should be started.
func MetricsEnabled() bool {
	return MetricsPort.Setting() != "disabled"
}

// ValidateSecureMetricsSetting returns an error if the environment variable is invalid.
func ValidateSecureMetricsSetting() error {
	if !SecureMetricsEnabled() {
		return nil
	}
	if err := validateTLS(); err != nil {
		return err
	}
	return validatePort(SecureMetricsPort)
}

// SecureMetricsEnabled returns true if the metrics/debug https server should be started.
func SecureMetricsEnabled() bool {
	return EnableSecureMetrics.BooleanSetting()
}
