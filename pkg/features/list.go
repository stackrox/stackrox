package features

var (
	// AuditLogging is used to enable the audit logging interceptor
	AuditLogging = registerFeature("Enables Audit logging", "ROX_AUDIT_LOGGING", true)

	// PerformDeploymentReconciliation controls whether we actually do the reconciliation.
	// It exists while we stabilize the feature.
	// NB: When removing this feature flag, please also remove references to it in .circleci/config.yml
	PerformDeploymentReconciliation = registerFeature("Reconciliation", "ROX_PERFORM_DEPLOYMENT_RECONCILIATION", true)

	// K8sRBAC is used to enable k8s rbac collection and processing
	K8sRBAC = registerFeature("Enable k8s RBAC objects collection and processing", "ROX_K8S_RBAC", false)

	// CentralTLSSecretLoader will read secrets from the orchestrator to configure TLS.
	CentralTLSSecretLoader = registerFeature("Enable Central TLS secret loading from orchestrator", "ROX_CENTRAL_TLS", false)

	// ProcessWhitelist will enable the process whitelist API
	ProcessWhitelist = registerFeature("Enable Process Whitelist API", "ROX_PROCESS_WHITELIST", false)
)
