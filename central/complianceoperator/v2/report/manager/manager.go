package manager

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Manager implements the interface to run report jobs
//
//go:generate mockgen-wrapper
type Manager interface {
	// SubmitReportRequest queues an on demand compliance report generation request for execution
	SubmitReportRequest(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, notificationMethod storage.ComplianceOperatorReportStatus_NotificationMethod) error

	// Start Scheduler
	Start()
	// Stop scheduler
	Stop()

	// OnScanConfigUpsert notifies the report scheduler that a scan
	// configuration was created or updated, so its scheduled reports can
	// start (or pick up the changes on the next wake-up). It is a no-op
	// unless reports run on their own scheduler
	// (ROX_COMPLIANCE_REPORTS_OWN_SCHEDULER) on this Central instance.
	OnScanConfigUpsert(scanConfig *storage.ComplianceOperatorScanConfigurationV2)

	// OnScanConfigDeleted notifies the report scheduler that a scan
	// configuration was deleted, so its scheduled reports stop.
	OnScanConfigDeleted(scanConfigID string)

	HandleScan(ctx context.Context, scan *storage.ComplianceOperatorScanV2) error
	HandleScanRemove(scanID string) error
	HandleResult(ctx context.Context, result *storage.ComplianceOperatorCheckResultV2) error
}
