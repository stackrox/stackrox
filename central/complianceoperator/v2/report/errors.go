package report

import "errors"

var (
	ErrReportGeneration           = errors.New("unable to generate the report")
	ErrSendingEmail               = errors.New("unable to send the report email")
	ErrUnableToSubscribeToWatcher = errors.New("unable to subscribe to scan watcher")
	ErrNoNotifiersConfigured      = errors.New("no notifiers configured")
	ErrScanWatchersFailed         = errors.New("scan watchers failed")
	ErrScanConfigWatcherTimeout   = errors.New("timeout waiting for the scans to finish")
)
