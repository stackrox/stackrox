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

	// OpenShiftClientCANamespace has the namespace of the OpenShift client CA ConfigMap
	// (extension-apiserver-authentication). Env var name is historical and shared with secure metrics.
	OpenShiftClientCANamespace = RegisterSetting("ROX_SECURE_METRICS_CLIENT_CA_NS", WithDefault("kube-system"))
	// OpenShiftClientCAConfigMap has the ConfigMap that contains the OpenShift client CA.
	OpenShiftClientCAConfigMap = RegisterSetting("ROX_SECURE_METRICS_CLIENT_CA_CFG", WithDefault("extension-apiserver-authentication"))
	// OpenShiftClientCAKey has the ConfigMap key that contains the OpenShift client CA.
	OpenShiftClientCAKey = RegisterSetting("ROX_SECURE_METRICS_CLIENT_CA_KEY", WithDefault("client-ca-file"))
	// OpenShiftClientCertCN has the expected subject CN of OpenShift platform clients (e.g. Prometheus).
	OpenShiftClientCertCN = RegisterSetting("ROX_SECURE_METRICS_CLIENT_CERT_CN", WithDefault("system:serviceaccount:openshift-monitoring:prometheus-k8s"))

	// OpenShiftTLSCertDir has the OpenShift service-serving certificate for the central-ocp Service.
	OpenShiftTLSCertDir = RegisterSetting("ROX_OPENSHIFT_TLS_CERT_DIR", WithDefault("/run/secrets/stackrox.io/ocp-tls"))
)

// MetricsEnabled returns true if the metrics/debug http server should be started.
func MetricsEnabled() bool {
	return MetricsPort.Setting() != "disabled"
}

// SecureMetricsEnabled returns true if the metrics/debug https server should be started.
func SecureMetricsEnabled() bool {
	return EnableSecureMetrics.BooleanSetting()
}
