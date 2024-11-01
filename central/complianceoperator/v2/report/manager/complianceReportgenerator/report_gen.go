package complianceReportgenerator

import (
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	benchmarksDS "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore"
	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	remediationDS "github.com/stackrox/rox/central/complianceoperator/v2/remediations/datastore"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	complianceRuleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/pkg/notifier"
)

// ReportGenerator interface is used to generate compliance report and send email notification.
//
//go:generate mockgen-wrapper
type ComplianceReportGenerator interface {
	// ProcessReportRequest will generate a csv report and send notification via email to attached scan config notifiers.
	ProcessReportRequest(req *ComplianceReportRequest) error
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
	}
}
