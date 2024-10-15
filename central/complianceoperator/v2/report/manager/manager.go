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
	SubmitReportRequest(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) error

	// Start Scheduler
	Start()
	// Stop scheduler
	Stop()

	HandleScan(ctx context.Context, scan *storage.ComplianceOperatorScanV2) error
	HandleResult(ctx context.Context, result *storage.ComplianceOperatorCheckResultV2) error
}
