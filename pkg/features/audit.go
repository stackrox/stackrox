package features

var (
	// AuditLogging is used to enable the audit logging interceptor
	AuditLogging = registerFeature("Enables Audit logging", "ROX_AUDIT_LOGGING", true)
)
