package env

var (
	// CentralProxyCertPath is the path to the TLS certificate for the proxy server.
	CentralProxyCertPath = RegisterSetting("ROX_CENTRAL_PROXY_CERT_PATH",
		WithDefault("/run/secrets/stackrox.io/proxy-tls/tls.crt"))

	// CentralProxyKeyPath is the path to the TLS private key for the proxy server.
	CentralProxyKeyPath = RegisterSetting("ROX_CENTRAL_PROXY_KEY_PATH",
		WithDefault("/run/secrets/stackrox.io/proxy-tls/tls.key"))

	// CentralProxyPort is the port on which the proxy server listens.
	CentralProxyPort = RegisterSetting("ROX_CENTRAL_PROXY_PORT", WithDefault(":9444"))

	// SecuredClusterMetricsEnabled opts this Sensor into proxying Central custom
	// metrics for the OpenShift Prometheus service account.
	SecuredClusterMetricsEnabled = RegisterBooleanSetting("ROX_SECURED_CLUSTER_METRICS_ENABLED", false)

	// CentralProxyEnabled opts Sensor into serving configured Central API proxy clients on OpenShift.
	CentralProxyEnabled = RegisterBooleanSetting("ROX_CENTRAL_PROXY_ENABLED", false)
)
