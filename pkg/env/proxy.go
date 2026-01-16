package env

var (
	// CentralProxyToken is the ROX API token used for authenticating proxy requests to Central.
	CentralProxyToken = RegisterSetting("ROX_CENTRAL_PROXY_TOKEN")

	// CentralProxyCertPath is the path to the TLS certificate for the proxy server.
	CentralProxyCertPath = RegisterSetting("ROX_CENTRAL_PROXY_CERT_PATH",
		WithDefault("/run/secrets/stackrox.io/proxy-tls/tls.crt"))

	// CentralProxyKeyPath is the path to the TLS private key for the proxy server.
	CentralProxyKeyPath = RegisterSetting("ROX_CENTRAL_PROXY_KEY_PATH",
		WithDefault("/run/secrets/stackrox.io/proxy-tls/tls.key"))

	// CentralProxyPort is the port on which the proxy server listens.
	CentralProxyPort = RegisterSetting("ROX_CENTRAL_PROXY_PORT", WithDefault(":9444"))
)
