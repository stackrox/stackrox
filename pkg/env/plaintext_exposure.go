package env

var (
	// PlaintextEndpoints specifies non-TLS endpoints to expose, if any.
	PlaintextEndpoints = RegisterSetting("ROX_PLAINTEXT_ENDPOINTS", AllowEmpty())

	// SecureEndpoints specifies TLS endpoints to expose, if any.
	SecureEndpoints = RegisterSetting("ROX_SECURE_ENDPOINTS", AllowEmpty())
)
