package env

var (
	// ProcessBaselineRisk toggles whether to process baseline risk
	ProcessBaselineRisk = RegisterBooleanSetting("ROX_PROCESS_BASELINE_RISK", true)
)
