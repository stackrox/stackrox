package env

var (
	// CACertFileEnv allows to pass a custom CA certificate file (path to a certificate file in PEM format).
	CACertFileEnv = RegisterSetting("ROX_CA_CERT_FILE")

	// ClientForceHTTP1Env configures the use of HTTP/1 for all connections (advanced; only use if you encounter connection issues).
	ClientForceHTTP1Env = RegisterBooleanSetting("ROX_CLIENT_FORCE_HTTP1", false)

	// DirectGRPCEnv configures the use of direct gRPC (advanced; only use if you encounter connection issues).
	DirectGRPCEnv = RegisterBooleanSetting("ROX_DIRECT_GRPC_CLIENT", false)

	// EndpointEnv specifies the central endpoint to use for commandline operations.
	EndpointEnv = RegisterSetting("ROX_ENDPOINT")

	// InsecureClientEnv enables insecure client connection options (DANGEROUS, USE WITH CAUTION).
	InsecureClientEnv = RegisterBooleanSetting("ROX_INSECURE_CLIENT", false)

	// InsecureClientSkipTLSVerifyEnv allows commandline clients to skip the TLS certificate validation.
	InsecureClientSkipTLSVerifyEnv = RegisterBooleanSetting("ROX_INSECURE_CLIENT_SKIP_TLS_VERIFY", false)

	// NoColorEnv disables commandline color output.
	NoColorEnv = RegisterBooleanSetting("ROX_NO_COLOR", false)

	// PasswordEnv specifies the central admin password to use for commandline operations.
	PasswordEnv = RegisterSetting("ROX_ADMIN_PASSWORD")

	// PlaintextEnv specifies whether the commandline operations should communicate over unencrypted channesl.
	PlaintextEnv = RegisterBooleanSetting("ROX_PLAINTEXT", false)

	// ServerEnv specifies the central server name to use for commandline operations.
	ServerEnv = RegisterSetting("ROX_SERVER_NAME")

	// TokenEnv is the variable that clients can source for commandline operations.
	TokenEnv = RegisterSetting("ROX_API_TOKEN")

	// ConfigDirEnv is the variable that clients can use for specifying the config location for commandline operations.
	ConfigDirEnv = RegisterSetting("ROX_CONFIG_DIR")
)
