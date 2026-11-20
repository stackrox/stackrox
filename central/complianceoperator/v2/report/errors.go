package report

import "errors"

const (
	COMPLIANCE_VERSION_ERROR             = "Compliance Operator Version is older than 1.6.0"
	COMPLIANCE_NOT_INSTALLED             = "Compliance Operator is not installed"
	INTERNAL_ERROR                       = "Internal Error"
	SCAN_NOT_APPLICABLE                  = "All scans were not applicable for this cluster"
	SCAN_REMOVED_FMT                     = "Scan %s was removed"
	SCAN_TIMEOUT_FMT                     = "Timeout waiting for scan %s to finish"
	SCAN_TIMEOUT_SENSOR_DISCONNECTED_FMT = "Timeout waiting for scan %s to finish (Sensor disconnect during the scan)"

	// Reasons produced by the settle-audit-snapshot report primitive
	// (ROX_COMPLIANCE_REPORTS_OWN_SCHEDULER). They reuse the validator's taxonomy:
	// the findings are derived at report time instead of at watcher-timeout time.
	SENSOR_DISCONNECTED      = "Sensor disconnected"
	SCAN_OVERDUE_FMT         = "Scan %s is overdue: it has not started within the expected schedule window (last start: %s)"
	SCAN_NEVER_STARTED_FMT   = "Scan %s is overdue: it has never started"
	SCAN_SETTLE_TIMEOUT_FMT  = "Timeout waiting for scan %s to finish before generating the report"
	NO_SCANS_OF_CONFIG_FOUND = "No scans of the scan configuration were found in this cluster"
)

var (
	ErrReportGeneration           = errors.New("unable to generate the report")
	ErrSendingEmail               = errors.New("unable to send the report email")
	ErrUnableToSubscribeToWatcher = errors.New("unable to subscribe to scan watcher")
	ErrNoNotifiersConfigured      = errors.New("no notifiers configured")
	ErrScanWatchersFailed         = errors.New("scan watchers failed")
	ErrScanConfigWatcherTimeout   = errors.New("timeout waiting for the scans to finish")
)
