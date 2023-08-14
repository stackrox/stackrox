package v2

import (
	"context"

	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	"github.com/stackrox/rox/generated/storage"
)

// Scheduler maintains the schedules for reports
//
//go:generate mockgen-wrapper
type Scheduler interface {
	// UpsertReportSchedule adds/updates the schedule at which reports for the given report config are executed.
	UpsertReportSchedule(reportConfig *storage.ReportConfiguration) error
	// RemoveReportSchedule removes the given report configuration from scheduled execution.
	RemoveReportSchedule(reportConfigID string)

	// CanSubmitReportRequest returns true if the given user can submit an on-demand report request for the given report configuration.
	CanSubmitReportRequest(user *storage.SlimUser, reportConfig *storage.ReportConfiguration) (bool, error)

	// SubmitReportRequest submits a report execution request. The report request can be either for an on demand report or a scheduled report.
	// If there is already a pending report request submitted by the same user for the same report config, this request will be denied.
	// However, there can be multiple pending report requests for same configuration by different users.
	SubmitReportRequest(ctx context.Context, request *reportGen.ReportRequest, reSubmission bool) (string, error)

	// CancelReportRequest cancels a report request that is still waiting in queue.
	// If the report is already being prepared or has completed execution, it cannot be cancelled.
	CancelReportRequest(ctx context.Context, reportID string) (bool, error)

	// Start scheduler. A scheduler instance can only be started once. It cannot be re-started once stopped.
	// This func will log errors if the scheduler fails to start.
	Start()
	// Stop scheduler
	Stop()
}
