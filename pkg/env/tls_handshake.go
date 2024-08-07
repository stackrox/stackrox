package env

import "time"

var (
	// TLSHandshakeTimeout defines time in which TLS handshake must be finished.
	TLSHandshakeTimeout = registerDurationSetting("ROX_TLS_HANDSHAKE_TIMEOUT", 2*time.Second)
)
