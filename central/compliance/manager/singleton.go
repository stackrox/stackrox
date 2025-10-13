package manager

import (
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/data"
	complianceDS "github.com/stackrox/rox/central/compliance/datastore"
	"github.com/stackrox/rox/central/compliance/standards"
	complianceOperatorCheckDS "github.com/stackrox/rox/central/complianceoperator/checkresults/datastore"
	complianceOperatorManager "github.com/stackrox/rox/central/complianceoperator/manager"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
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
		managerInstance = NewManager(standards.RegistrySingleton(), complianceOperatorManager.Singleton(), complianceOperatorCheckDS.Singleton(), datastore.Singleton(), nodeDatastore.Singleton(), deploymentDatastore.Singleton(), podDatastore.Singleton(), data.NewDefaultFactory(), factory.Singleton(), complianceDS.Singleton())
	})
	return managerInstance
}
