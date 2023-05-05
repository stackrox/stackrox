package notifiers

import (
	"bytes"
	"context"

	"github.com/stackrox/rox/pkg/notifiers"
)

// ReportNotifier is a notifier for sending reports
//
//go:generate mockgen-wrapper ReportNotifier
type ReportNotifier interface {
	notifiers.Notifier
	// ReportNotify triggers the plugins to send a notification about a report
	ReportNotify(ctx context.Context, zippedReportData *bytes.Buffer, recipients []string, messageText string) error
}
