package manager

import (
	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	complianceIntegrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	snapshotDataStore "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	reportGen "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/generator"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	bindingsDS "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
	suiteDS "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Manager
)

// Singleton provides the instance of compliance report Manager to use.
func Singleton() Manager {
	once.Do(initialize)
	return instance
}

func initialize() {
	instance = New(scanConfigurationDS.Singleton(),
		scanDS.Singleton(),
		profileDatastore.Singleton(),
		snapshotDataStore.Singleton(),
		complianceIntegrationDS.Singleton(),
		suiteDS.Singleton(),
		bindingsDS.Singleton(),
		checkResults.Singleton(),
		reportGen.Singleton())
}
