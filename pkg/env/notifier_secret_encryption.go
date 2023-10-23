package env

var (
	// EncNotifierCreds controls if notifier creds encryption
	EncNotifierCreds = RegisterBooleanSetting("ROX_ENCRYPT_NOTIFIER_SECRETS", false)

	// CleanupNotifierCreds controls the cleanup of secrets
	CleanupNotifierCreds = RegisterBooleanSetting("ROX_CLEANUP_NOTIFIER_SECRETS", false)
)
