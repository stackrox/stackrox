package env

var (
	// ProcessFilterMaxProcessPaths sets the maximum number of process filter unique paths
	ProcessFilterMaxProcessPaths = RegisterIntegerSetting("ROX_PROCESS_FILTER_MAX_PROCESS_PATHS", 5000)
)
