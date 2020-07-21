package env

var (
	// OfflineModeEnv is the variable to ensure that StackRox doesn't reach out to the internet
	OfflineModeEnv = RegisterBooleanSetting("ROX_OFFLINE_MODE", false)
)
