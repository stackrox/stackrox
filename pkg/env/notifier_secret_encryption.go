package env

var (
	// EncNotifierCreds controls if notifier creds encryption
	EncNotifierCreds = RegisterBooleanSetting("ROX_ENC_NOTIFIER_CREDS", false)

	// CleanupNotifierCreds controls the cleanup of secrets
	CleanupNotifierCreds = RegisterBooleanSetting("ROX_CLEANUP_NOTIFIER_CREDS", false)
)
