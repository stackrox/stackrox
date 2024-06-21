package env

var (
	// EnableErrorConcealement configures central to drop messages of errors, not
	// wrapped with errox.SensitiveError.
	EnableErrorConcealement = RegisterBooleanSetting("ROX_ENABLE_ERROR_CONCEALEMENT", false)
)
