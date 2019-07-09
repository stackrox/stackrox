package env

var (
	// PlaintextEndpoints specifies non-TLS endpoints to expose, if any.
	PlaintextEndpoints = RegisterSetting("ROX_PLAINTEXT_ENDPOINTS", AllowEmpty())
)
