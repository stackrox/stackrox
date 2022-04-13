package manager

import (
	"github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/compliance/data"
	complianceDS "github.com/stackrox/stackrox/central/compliance/datastore"
	"github.com/stackrox/stackrox/central/compliance/standards"
	complianceOperatorCheckDS "github.com/stackrox/stackrox/central/complianceoperator/checkresults/datastore"
	complianceOperatorManager "github.com/stackrox/stackrox/central/complianceoperator/manager"
	deploymentDatastore "github.com/stackrox/stackrox/central/deployment/datastore"
	nodeDatastore "github.com/stackrox/stackrox/central/node/globaldatastore"
	podDatastore "github.com/stackrox/stackrox/central/pod/datastore"
	"github.com/stackrox/stackrox/central/scrape/factory"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	managerInstance     ComplianceManager
	managerInstanceInit sync.Once
)

// Singleton returns the compliance manager singleton instance.
func Singleton() ComplianceManager {
	managerInstanceInit.Do(func() {
		var err error
		managerInstance, err = NewManager(standards.RegistrySingleton(), complianceOperatorManager.Singleton(), complianceOperatorCheckDS.Singleton(), ScheduleStoreSingleton(), datastore.Singleton(), nodeDatastore.Singleton(), deploymentDatastore.Singleton(), podDatastore.Singleton(), data.NewDefaultFactory(), factory.Singleton(), complianceDS.Singleton())
		if err != nil {
			log.Panicf("Could not create compliance manager: %v", err)
		}
	})
	return managerInstance
}
