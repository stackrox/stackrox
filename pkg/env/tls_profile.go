package env

var (
	// TLSMinVersion configures the minimum TLS version for ACS services.
	// Accepted values: "TLSv1.2", "TLSv1.3".
	// When unset, services default to TLS 1.2.
	TLSMinVersion = RegisterSetting("ROX_TLS_MIN_VERSION")

	// TLSCipherSuites configures the allowed TLS cipher suites for ACS services.
	// Value is a comma-separated list of IANA cipher suite names
	// (e.g. "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384").
	// Only affects TLS 1.2 negotiation; TLS 1.3 cipher suites are fixed by Go
	// and not configurable.
	// When unset, services use their compiled-in defaults.
	TLSCipherSuites = RegisterSetting("ROX_TLS_CIPHER_SUITES")
)
