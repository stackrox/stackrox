package manager

import (
	"context"
)

// Manager implements the interface to run report jobs
//
//go:generate mockgen-wrapper
type Manager interface {
	// SubmitReportRequest queues an on demand report generation request for execution
	SubmitReportRequest(ctx context.Context, scanConfigID string) error
}
