package env

var (
	// LoggingMaxBackups is the maximum number of old log files to retain.
	// The default is to retain 5 old log files.
	LoggingMaxBackups = RegisterIntegerSetting("ROX_LOGGING_MAX_BACKUPS", 5)

	// LoggingMaxSizeMB is the maximum size in megabytes of the log file before
	// it gets rotated. It defaults to 20 megabytes.
	LoggingMaxSizeMB = RegisterIntegerSetting("ROX_LOGGING_MAX_SIZE_MB", 20)
)
