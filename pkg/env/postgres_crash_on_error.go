package env

var (
	// PostgresCrashOnError crashes if a retryable error exceeds its retries
	PostgresCrashOnError = RegisterBooleanSetting("ROX_POSTGRES_CRASH_ON_ERROR", true)
)
