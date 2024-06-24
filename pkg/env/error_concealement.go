package env

var (
	// EnableErrorConcealment configures central to drop messages of errors, not
	// wrapped with errox.SensitiveError.
	EnableErrorConcealment = RegisterBooleanSetting("ROX_ENABLE_ERROR_CONCEALMENT", false)
)
