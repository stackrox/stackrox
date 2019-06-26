package manager

import (
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/data"
	complianceDS "github.com/stackrox/rox/central/compliance/datastore"
	"github.com/stackrox/rox/central/compliance/standards"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	nodeDatastore "github.com/stackrox/rox/central/node/globaldatastore"
	"github.com/stackrox/rox/central/scrape/factory"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	managerInstance     ComplianceManager
	managerInstanceInit sync.Once
)

// Singleton returns the compliance manager singleton instance.
func Singleton() ComplianceManager {
	managerInstanceInit.Do(func() {
		var err error
		managerInstance, err = NewManager(standards.RegistrySingleton(), ScheduleStoreSingleton(), datastore.Singleton(), nodeDatastore.Singleton(), deploymentDatastore.Singleton(), data.NewDefaultFactory(), factory.Singleton(), complianceDS.Singleton())
		if err != nil {
			log.Panicf("Could not create compliance manager: %v", err)
		}
	})
	return managerInstance
}
