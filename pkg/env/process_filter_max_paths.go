package env

var (
	// ProcessFilterMaxProcessPaths sets the maximum number of process filter unique paths.
	// This setting can be overridden by ROX_PROCESS_FILTER_MODE presets.
	// Default: 5000
	ProcessFilterMaxProcessPaths = RegisterIntegerSetting("ROX_PROCESS_FILTER_MAX_PROCESS_PATHS", 5000)
)
