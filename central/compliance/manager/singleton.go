package manager

import (
	"sync"

	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/data"
	complianceResultsStore "github.com/stackrox/rox/central/compliance/store"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	nodeStore "github.com/stackrox/rox/central/node/globalstore"
	"github.com/stackrox/rox/central/scrape"
)

var (
	managerInstance     ComplianceManager
	managerInstanceInit sync.Once
)

// Singleton returns the compliance manager singleton instance.
func Singleton() ComplianceManager {
	managerInstanceInit.Do(func() {
		var err error
		managerInstance, err = NewManager(DefaultStandardImplementationStore(), ScheduleStoreSingleton(), datastore.Singleton(), nodeStore.Singleton(), deploymentDatastore.Singleton(), data.NewDefaultFactory(), scrape.SingletonFactory(), complianceResultsStore.Singleton())
		if err != nil {
			log.Panicf("Could not create compliance manager: %v", err)
		}
	})
	return managerInstance
}
