package env

var (
	// PlaintextPort specifies the non-TLS port, if any.
	PlaintextPort = RegisterSetting("ROX_PLAINTEXT_PORT", AllowEmpty())
)
