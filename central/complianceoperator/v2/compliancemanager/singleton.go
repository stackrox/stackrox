package compliancemanager

import (
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	resultsDatastore "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	compIntegration "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	compScanSetting "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scansDatastore "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	manager Manager
	once    sync.Once
)

// Singleton returns the compliance operator manager
func Singleton() Manager {
	once.Do(func() {
		manager = New(connection.ManagerSingleton(), compIntegration.Singleton(), compScanSetting.Singleton(), clusterDatastore.Singleton(), profileDatastore.Singleton(), scansDatastore.Singleton(), resultsDatastore.Singleton())
	})
	return manager
}
