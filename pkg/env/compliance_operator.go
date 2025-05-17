package env

import "time"

var (
	// ComplianceScanTimeout defines the timeout for compliance scan. If the scan hasn't finished by then, it will be aborted.
	// Compliance operator default is 30 mins.
	ComplianceScanTimeout = registerDurationSetting("ROX_COMPLIANCE_SCAN_TIMEOUT", 15*time.Minute)
	// ComplianceScanRetries defines is the maximum number of times the scan will be retried if it times out.
	// Compliance operator default is 3.
	ComplianceScanRetries = RegisterIntegerSetting("ROX_COMPLIANCE_SCAN_RETRY_COUNT", 2)
	// ComplianceStrictNodeScan defines if scans can proceed if the scan should fail if any node cannot be scanned
	ComplianceStrictNodeScan = RegisterBooleanSetting("ROX_COMPLIANCE_STRICT_NODE_SCAN", true)

	// ComplianceScanWatcherTimeout defines the timeout for a compliance scan watcher.
	// If the scan results have not been received by then, it will be aborted.
	// The default is 40 mins.
	ComplianceScanWatcherTimeout = registerDurationSetting("ROX_COMPLIANCE_SCAN_WATCHER_TIMEOUT", 40*time.Minute)

	// ComplianceScanScheduleWatcherTimeout defines the timeout for a compliance scan schedule watcher.
	// If the scan results of all scans associated with the schedule have not been received by then, it will be aborted.
	// The default is 45 mins.
	ComplianceScanScheduleWatcherTimeout = registerDurationSetting("ROX_COMPLIANCE_SCAN_SCHEDULE_WATCHER_TIMEOUT", 45*time.Minute)

	// ComplianceMaxNumberOfErrorsInReport defines the max number of errors that a report will store. This is done to avoid overwhelming the UI with many errors.
	// The default is 4
	ComplianceMaxNumberOfErrorsInReport = RegisterIntegerSetting("ROX_COMPLIANCE_MAX_NUMBER_OF_ERRORS_IN_REPORT", 4)

	// ComplianceMinimalSupportedVersion specifies the minimum version of the compliance operator that is supported by StackRox.
	// This value can be customized via the ROX_COMPLIANCE_MINIMAL_SUPPORTED_OPERATOR_VERSION environment variable.
	// If the environment variable is unset or contains an invalid version, the default value "v1.6.0" will be used.
	ComplianceMinimalSupportedVersion = RegisterVersionSetting("ROX_COMPLIANCE_MINIMAL_SUPPORTED_OPERATOR_VERSION", "v1.6.0")
)
