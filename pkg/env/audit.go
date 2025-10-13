package env

var (
	// AuditLogWithoutPermissions allows to alter the message sent for audit logs to _not_ include detailed permissions
	// for the user.
	AuditLogWithoutPermissions = RegisterBooleanSetting("ROX_AUDIT_LOG_WITHOUT_PERMISSIONS", false)
)
