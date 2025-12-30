package env

var (
	// ProcessFilterMaxExactPathMatches sets the maximum number of times an exact
	// process path (same deployment+container+process+args) can appear before being filtered
	ProcessFilterMaxExactPathMatches = RegisterIntegerSetting(
		"ROX_PROCESS_FILTER_MAX_EXACT_PATH_MATCHES",
		5,
	).WithMinimum(1).WithMaximum(1000)

	// ProcessFilterFanOutLevels sets the fan-out limits for each argument level.
	// Format: comma-separated integers (e.g., "8,6,4,2")
	// Each value represents the maximum number of unique children allowed at that level.
	// An empty string results in an empty array, which means only unique processes are tracked without argument tracking.
	ProcessFilterFanOutLevels = RegisterIntegerArraySetting(
		"ROX_PROCESS_FILTER_FAN_OUT_LEVELS",
		[]int{8, 6, 4, 2},
	).WithMinimumValue(1).WithMaximumValue(1000).WithMinLength(0).WithMaxLength(10)
)
