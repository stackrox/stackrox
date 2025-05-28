package report

import "errors"

const (
	COMPLIANCE_VERSION_ERROR             = "Compliance Operator Version is older than 1.6.0"
	COMPLIANCE_NOT_INSTALLED             = "Compliance Operator is not installed"
	INTERNAL_ERROR                       = "Internal Error"
	SCAN_REMOVED_FMT                     = "Scan %s was removed"
	SCAN_TIMEOUT_FMT                     = "Timeout waiting for scan %s to finish"
	SCAN_TIMEOUT_SENSOR_DISCONNECTED_FMT = "Timeout waiting for scan %s to finish (Sensor disconnect during the scan)"
)

var (
	ErrReportGeneration           = errors.New("unable to generate the report")
	ErrSendingEmail               = errors.New("unable to send the report email")
	ErrUnableToSubscribeToWatcher = errors.New("unable to subscribe to scan watcher")
	ErrNoNotifiersConfigured      = errors.New("no notifiers configured")
	ErrScanWatchersFailed         = errors.New("scan watchers failed")
	ErrScanConfigWatcherTimeout   = errors.New("timeout waiting for the scans to finish")
)
