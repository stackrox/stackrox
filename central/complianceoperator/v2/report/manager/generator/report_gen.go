package generator

import (
	"bytes"
	"context"

	blobDS "github.com/stackrox/rox/central/blob/datastore"
	benchmarksDS "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore"
	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	remediationDS "github.com/stackrox/rox/central/complianceoperator/v2/remediations/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/format"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/results"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/sender"
	complianceRuleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/notifier"
)

// ReportGenerator interface is used to generate compliance report and send email notification.
//
//go:generate mockgen-wrapper
type ComplianceReportGenerator interface {
	// ProcessReportRequest will generate a csv report and send notification via email to attached scan config notifiers.
	ProcessReportRequest(req *report.Request) error
	// Stop will stop all the sender watchers
	Stop()
}

// Formatter interface is used to generate the report zip file containing the csv files
//
//go:generate mockgen-wrapper
type Formatter interface {
	FormatCSVReport(map[string][]*report.ResultRow, map[string]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster) (*bytes.Buffer, error)
}

// ResultsAggregator interface is used to generate the report data
//
//go:generate mockgen-wrapper
type ResultsAggregator interface {
	GetReportData(*report.Request) *report.Results
}

//go:generate mockgen-wrapper
type ReportSender interface {
	SendEmail(context.Context, string, *bytes.Buffer, *report.Results, []*storage.NotifierConfiguration) <-chan error
}

// New will create a new instance of the ReportGenerator
func New(checkResultDS checkResults.DataStore, notifierProcessor notifier.Processor, profileDS profileDS.DataStore, remediationDS remediationDS.DataStore, scanDS scanDS.DataStore, benchmarksDS benchmarksDS.DataStore, complianceRuleDS complianceRuleDS.DataStore, snapshotDS snapshotDS.DataStore, blobDS blobDS.Datastore) ComplianceReportGenerator {
	return &complianceReportGeneratorImpl{
		checkResultsDS:           checkResultDS,
		notificationProcessor:    notifierProcessor,
		profileDS:                profileDS,
		remediationDS:            remediationDS,
		scanDS:                   scanDS,
		benchmarkDS:              benchmarksDS,
		complianceRuleDS:         complianceRuleDS,
		snapshotDS:               snapshotDS,
		blobStore:                blobDS,
		numberOfTriesOnEmailSend: defaultNumberOfTriesOnEmailSend,
		formatter:                format.NewFormatter(),
		resultsAggregator:        results.NewAggregator(checkResultDS, scanDS, profileDS, remediationDS, benchmarksDS, complianceRuleDS),
		reportSender:             sender.NewReportSender(notifierProcessor, defaultNumberOfTriesOnEmailSend),
		senderResponseHandlers:   make(map[string]stoppable[error]),
		newHandlerFn:             sender.NewAsyncResponseHandler[error],
	}
}
