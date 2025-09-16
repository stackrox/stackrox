package env

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
	// SecureMetricsClientCAKey has the config map key that contains the client CA.
	SecureMetricsClientCAKey = RegisterSetting("ROX_SECURE_METRICS_CLIENT_CA_KEY", WithDefault("client-ca-file"))
	// SecureMetricsClientCertCN has the expected subject common name of the client cert.
	SecureMetricsClientCertCN = RegisterSetting("ROX_SECURE_METRICS_CLIENT_CERT_CN", WithDefault("system:serviceaccount:openshift-monitoring:prometheus-k8s"))
)

// MetricsEnabled returns true if the metrics/debug http server should be started.
func MetricsEnabled() bool {
	return MetricsPort.Setting() != "disabled"
}

// SecureMetricsEnabled returns true if the metrics/debug https server should be started.
func SecureMetricsEnabled() bool {
	return EnableSecureMetrics.BooleanSetting()
}
