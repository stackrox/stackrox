package service

import (
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/compliancemanager"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	reportManager "github.com/stackrox/rox/central/complianceoperator/v2/report/manager"
	reportScheduler "github.com/stackrox/rox/central/complianceoperator/v2/report/scheduler"
	scanSettingsDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanSettingBindingsDS "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
	suiteDS "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore"
	"github.com/stackrox/rox/central/globaldb"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	serviceInstance     Service
	serviceInstanceInit sync.Once
)

// Singleton returns the singleton instance of the compliance service.
func Singleton() Service {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}

	if features.ComplianceReporting.Enabled() {
		go reportManager.Singleton().Start()
	}

	serviceInstanceInit.Do(func() {
		// Started here rather than on every call: the decoupled report scheduler holds a
		// PostgreSQL advisory lock while running, which a second start could not re-acquire.
		// Start itself is a no-op while ROX_COMPLIANCE_DECOUPLED_REPORTS is disabled.
		if features.ComplianceReporting.Enabled() {
			go reportScheduler.Singleton().Start(globaldb.GetPostgres())
		}
		serviceInstance = New(scanSettingsDS.Singleton(), scanSettingBindingsDS.Singleton(), suiteDS.Singleton(),
			compliancemanager.Singleton(), reportManager.Singleton(), reportScheduler.Singleton(), notifierDS.Singleton(), profileDS.Singleton(), clusterDatastore.Singleton(),
			snapshotDS.Singleton(), blobDS.Singleton())
	})
	return serviceInstance
}
