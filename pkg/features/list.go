package features

var (
	// AuditLogging is used to enable the audit logging interceptor
	AuditLogging = registerFeature("Enables Audit logging", "ROX_AUDIT_LOGGING", true)

	// K8sRBAC is used to enable k8s rbac collection and processing
	// NB: When removing this feature flag, please also remove references to it in .circleci/config.yml
	K8sRBAC = registerFeature("Enable k8s RBAC objects collection and processing", "ROX_K8S_RBAC", true)

	// ProcessWhitelist will enable the process whitelist API
	// NB: When removing this feature flag, please also remove references to it in .circleci/config.yml
	ProcessWhitelist = registerFeature("Enable Process Whitelist API", "ROX_PROCESS_WHITELIST", true)

	// ScopedAccessControl controls whether scoped access control is enabled.
	// NB: When removing this feature flag, please also remove references to it in .circleci/config.yml
	ScopedAccessControl = registerFeature("Scoped Access Control", "ROX_SCOPED_ACCESS_CONTROL", false)

	// ClientCAAuth will enable authenticating to central via client certificate authentication
	ClientCAAuth = registerFeature("Client Certificate Authentication", "ROX_CLIENT_CA_AUTH", false)
)
