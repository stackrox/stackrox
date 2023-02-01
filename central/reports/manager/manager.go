package manager

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Manager implements the interface for scheduled reports
//
//go:generate mockgen-wrapper
type Manager interface {
	// Upsert adds/updates a report configuration into the scheduler
	Upsert(ctx context.Context, rc *storage.ReportConfiguration) error
	// Remove removes a report configuration from the scheduler and from future scheduled runs
	Remove(ctx context.Context, id string) error
	// RunReport queues an on demand report generation request for execution
	RunReport(ctx context.Context, rc *storage.ReportConfiguration) error
	Start()
	Stop()
}
