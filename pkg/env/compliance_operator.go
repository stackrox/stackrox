package env

import "time"

var (
	// ComplianceScanTimeout defines the timeout for compliance scan. If the scan hasn't finished by then, it will be aborted.
	// Compliance operator default is 30 mins.
	ComplianceScanTimeout = registerDurationSetting("ROX_COMPLIANCE_SCAN_TIMEOUT", 30*time.Minute)
	// ComplianceScanRetries defines is the maximum number of times the scan will be retried if it times out.
	// Compliance operator default is 3.
	ComplianceScanRetries = RegisterIntegerSetting("ROX_COMPLIANCE_SCAN_RETRY_COUNT", 3)
)
