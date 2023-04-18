package env

var (
	// CACertFileEnv allows to pass a custom CA certificate file (path to a certificate file in PEN format).
	CACertFileEnv = RegisterSetting("ROX_CA_CERT_FILE")

	// ClientForceHTTP1Env ...
	ClientForceHTTP1Env = RegisterBooleanSetting("ROX_CLIENT_FORCE_HTTP1", false)

	// DirectGRPCEnv ...
	DirectGRPCEnv = RegisterBooleanSetting("ROX_DIRECT_GRPC_CLIENT", false)

	// EndpointEnv specifies the central endpoint to use for commandline operations.
	EndpointEnv = RegisterSetting("ROX_ENDPOINT")

	// InsecureClientEnv enables insecure client connection options (DANGEROUS, USE WITH CAUTION).
	InsecureClientEnv = RegisterBooleanSetting("ROX_INSECURE_CLIENT", false)

	// InsecureClientSkipTLSVerifyEnv allows commandline clients to skip the TLS certificate validation.
	InsecureClientSkipTLSVerifyEnv = RegisterBooleanSetting("ROX_INSECURE_CLIENT_SKIP_TLS_VERIFY", false)

	// NoColorPrinterEnv disables commandline color output.
	NoColorPrinterEnv = RegisterBooleanSetting("ROX_NO_COLOR_PRINTER", false)

	// PasswordEnv specifies the central admin password to use for commandline operations.
	PasswordEnv = RegisterSetting("ROX_ADMIN_PASSWORD")

	// PlaintextEnv specifies whether the commandline operations should communicate over unencrypted channesl.
	PlaintextEnv = RegisterBooleanSetting("ROX_PLAINTEXT", false)

	// ServerEnv specifies the central server name to use for commandline operations.
	ServerEnv = RegisterSetting("ROX_SERVER_NAME")

	// TokenEnv is the variable that clients can source for commandline operations.
	TokenEnv = RegisterSetting("ROX_API_TOKEN")
)
