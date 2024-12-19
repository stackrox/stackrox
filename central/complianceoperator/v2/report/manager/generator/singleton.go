package generator

import (
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	benchmarksDS "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore"
	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	remediationDS "github.com/stackrox/rox/central/complianceoperator/v2/remediations/datastore"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	complianceRuleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance ComplianceReportGenerator
)

// Singleton provides the instance of Manager to use.
func Singleton() ComplianceReportGenerator {
	once.Do(initialize)
	return instance
}

func initialize() {
	instance = New(checkResults.Singleton(), notifierProcessor.Singleton(), profileDS.Singleton(), remediationDS.Singleton(), scanDS.Singleton(), benchmarksDS.Singleton(), complianceRuleDS.Singleton(), snapshotDS.Singleton(), blobDS.Singleton())
}
