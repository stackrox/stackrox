package env

var (
	// LoggingMaxRotationFiles is the maximum number of log rotation files
	// to retain. The default is to retain 5 rotation files in addition to the
	// current log file.
	LoggingMaxRotationFiles = RegisterIntegerSetting("ROX_LOGGING_MAX_ROTATION_FILES", 5)

	// LoggingMaxSizeMB is the maximum size in megabytes of the log file before
	// it gets rotated. It defaults to 20 megabytes.
	LoggingMaxSizeMB = RegisterIntegerSetting("ROX_LOGGING_MAX_SIZE_MB", 20)
)
