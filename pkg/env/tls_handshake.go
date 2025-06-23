package env

import "time"

var (
	// TLSHandshakeTimeout defines time in which TLS handshake must be finished.
	TLSHandshakeTimeout = registerDurationSetting("ROX_TLS_HANDSHAKE_TIMEOUT", 2*time.Second)
	// ForceServerALPNProtocols is comma-separated list of custom ALPN protocols advertised by the server.
	ForceServerALPNProtocols = RegisterSetting("ROX_TLS_FORCE_SERVER_ALPN_PROTOCOLS", WithDefault(""))
	// ForceClientALPNProtocols is comma-separated list of custom ALPN protocols advertised by the client.
	ForceClientALPNProtocols = RegisterSetting("ROX_TLS_FORCE_CLIENT_ALPN_PROTOCOLS", WithDefault(""))
)
