package env

var (
	// ReportExecutionMaxConcurrency sets the maximum number vulnerability reports that can run in parallel
	ReportExecutionMaxConcurrency = RegisterIntegerSetting("ROX_REPORT_EXECUTION_MAX_CONCURRENCY", 5)

	// VulnReportingEnhancements enables APIs and UI pages for VM Reporting enhancements including downloadable reports
	VulnReportingEnhancements = RegisterBooleanSetting("ROX_VULN_MGMT_REPORTING_ENHANCEMENTS", false)
)
