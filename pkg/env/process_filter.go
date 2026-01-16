package env

var (
	// ProcessFilterMaxExactPathMatches sets the maximum number of times an exact
	// process path (same deployment+container+process+args) can appear before being filtered
	ProcessFilterMaxExactPathMatches = RegisterIntegerSetting(
		"ROX_PROCESS_FILTER_MAX_EXACT_PATH_MATCHES",
		5,
	).WithMinimum(1).WithMaximum(1000)

	// ProcessFilterFanOutLevels sets the fan-out limits for each argument level.
	// Format: comma-separated integers within brackets (e.g., "[8,6,4,2]")
	// Each value represents the maximum number of unique children allowed at that level.
	// An empty value "" results in the default value being used.
	// An empty array "[]" means only unique processes are tracked without argument tracking.
	ProcessFilterFanOutLevels = RegisterIntegerArraySetting(
		"ROX_PROCESS_FILTER_FAN_OUT_LEVELS",
		[]int{8, 6, 4, 2},
	).WithMinimumElementValue(1).WithMaximumElementValue(1000).WithMinLength(0).WithMaxLength(10)
)
