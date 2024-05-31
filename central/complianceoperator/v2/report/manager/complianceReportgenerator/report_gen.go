package complianceReportgenerator

import (
<<<<<<< HEAD
	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
=======
	"bytes"
	"context"

	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	complianceRulesDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	"github.com/stackrox/rox/generated/storage"
>>>>>>> 6faeddcd64 (Added test file)
	"github.com/stackrox/rox/pkg/notifier"
)

// ReportGenerator interface is used to generate compliance report and send email notification.
//
//go:generate mockgen-wrapper
type ComplianceReportGenerator interface {
	// ProcessReportRequest will generate a csv report and send notification via email to attached scan config notifiers.
<<<<<<< HEAD
	ProcessReportRequest(req *ComplianceReportRequest)
=======
	ProcessReportRequest(ctx context.Context, req *ComplianceReportRequest) error

	getDataforReport(req *ComplianceReportRequest, ctx context.Context) (*resultEmail, error)

	sendEmail(zipData *bytes.Buffer, emailBody *formatBody, formatEmailSub *formatSubject, notifiersList []*storage.NotifierConfiguration, ctx context.Context) error

	Format(results map[string][]*resultRow) (*bytes.Buffer, error)
>>>>>>> 6faeddcd64 (Added test file)
}

// New will create a new instance of the ReportGenerator
func New(checkResultDS checkResults.DataStore, notifierProcessor notifier.Processor, ruleDS complianceRulesDS.DataStore, profileDS profileDS.DataStore) ComplianceReportGenerator {
	return &complianceReportGeneratorImpl{
		checkResultsDS:        checkResultDS,
		notificationProcessor: notifierProcessor,
		rulesDS:               ruleDS,
		profileDS:             profileDS,
	}
}
