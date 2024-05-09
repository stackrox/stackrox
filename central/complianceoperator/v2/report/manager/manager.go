package manager

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Manager implements the interface to run report jobs
//
//go:generate mockgen-wrapper
type Manager interface {
	// SubmitReportRequest queues an on demand report generation request for execution
	SubmitReportRequest(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) error

	Start()
	Stop()
}
