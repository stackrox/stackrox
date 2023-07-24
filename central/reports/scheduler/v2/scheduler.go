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
	UpsertReportSchedule(reportConfig *storage.ReportConfiguration) error
	RemoveReportSchedule(reportConfigID string)
	SubmitReportRequest(request *reportGen.ReportRequest, reSubmission bool) (string, error)
	CancelReportRequest(ctx context.Context, reportID string) (bool, string, error)
	Start()
	Stop()
}
