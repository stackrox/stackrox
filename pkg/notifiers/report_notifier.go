package notifiers

import (
	"bytes"
	"context"
)

// ReportNotifier is a notifier for sending reports
//
//go:generate mockgen-wrapper ReportNotifier
type ReportNotifier interface {
	Notifier
	// ReportNotify triggers the plugins to send a notification about a report
	ReportNotify(ctx context.Context, zippedReportData *bytes.Buffer, recipients []string, subject, messageText string) error
}
