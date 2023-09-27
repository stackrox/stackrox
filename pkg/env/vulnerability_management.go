package env

var (
	// ReportExecutionMaxConcurrency sets the maximum number vulnerability reports that can run in parallel
	ReportExecutionMaxConcurrency = RegisterIntegerSetting("ROX_REPORT_EXECUTION_MAX_CONCURRENCY", 5)
)
