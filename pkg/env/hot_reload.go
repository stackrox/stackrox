package env

var (
	// HotReload specifies if the binary is currently being hot reloaded
	HotReload = RegisterBooleanSetting("ROX_HOTRELOAD", false)
)
