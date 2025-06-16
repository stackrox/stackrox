package env

import "time"

var (
	// TLSHandshakeTimeout defines time in which TLS handshake must be finished.
	TLSHandshakeTimeout = registerDurationSetting("ROX_TLS_HANDSHAKE_TIMEOUT", 2*time.Second)
	// CustomALPNProtocols is comma-separated list of custom ALPN protocols advertised by the client.
	CustomALPNProtocols = RegisterSetting("ROX_TLS_ALPN_PROTOCOLS", WithDefault("https://alpn.stackrox.io/#pure-grpc"))
)
