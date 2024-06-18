package complianceReportgenerator

import (
	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	"github.com/stackrox/rox/pkg/notifier"
)

// ReportGenerator interface is used to generate compliance report and send email notification.
//
//go:generate mockgen-wrapper
type ComplianceReportGenerator interface {
	// ProcessReportRequest will generate a csv report and send notification via email to attached scan config notifiers.
	ProcessReportRequest(req *ComplianceReportRequest)
}

// New will create a new instance of the ReportGenerator
func New(checkResultDS checkResults.DataStore, notifierProcessor notifier.Processor) ComplianceReportGenerator {
	return &complianceReportGeneratorImpl{
		checkResultsDS:        checkResultDS,
		notificationProcessor: notifierProcessor,
	}
}
