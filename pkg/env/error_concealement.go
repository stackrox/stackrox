package env

var (
	// EnableErrorConcealment configures central to drop messages of errors, not
	// wrapped with errox.WithUserMessage.
	EnableErrorConcealment = RegisterBooleanSetting("ROX_ENABLE_ERROR_CONCEALMENT", false)
)
