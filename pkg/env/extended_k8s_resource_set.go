package env

var (
	// AuditPolicyExtendedSet enables policies with an extended set of resource types to be created for k8s events
	AuditPolicyExtendedSet = RegisterBooleanSetting("ROX_AUDIT_POLICY_EXTENDED_SET", false)
)
