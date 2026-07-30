package env

var (
	// ProcessPruningEnabled toggles whether process pruning should be done periodically using heuristics
	// This may be useful in certain large environments, but is fairly expensive as it requires a full
	// sweep over the database
	ProcessPruningEnabled = RegisterBooleanSetting("ROX_PROCESS_PRUNING", false)

	// ProcessPruningExperiment enables shadow-table instrumentation that records
	// both the old Jaccard and new normalized-exact-match pruning decisions without
	// affecting the actual pruning behavior. Requires ROX_PROCESS_PRUNING=true.
	ProcessPruningExperiment = RegisterBooleanSetting("ROX_PROCESS_PRUNING_EXPERIMENT", false)
)
