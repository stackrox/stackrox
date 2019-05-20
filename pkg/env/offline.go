package env

var (
	// OfflineModeEnv is the variable to ensure that StackRox doesn't reach out to the internet
	OfflineModeEnv = RegisterSetting("ROX_OFFLINE_MODE", WithDefault("false"))
)
