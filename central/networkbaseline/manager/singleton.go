package manager

import (
	deploymentDS "github.com/stackrox/stackrox/central/deployment/datastore"
	"github.com/stackrox/stackrox/central/networkbaseline/datastore"
	networkEntityDS "github.com/stackrox/stackrox/central/networkgraph/entity/datastore"
	nfDS "github.com/stackrox/stackrox/central/networkgraph/flow/datastore"
	networkPolicyDS "github.com/stackrox/stackrox/central/networkpolicies/datastore"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once     sync.Once
	instance Manager
)

// Singleton provides the instance of Manager to use.
func Singleton() Manager {
	once.Do(func() {
		var err error
		instance, err =
			New(
				datastore.Singleton(),
				networkEntityDS.Singleton(),
				deploymentDS.Singleton(),
				networkPolicyDS.Singleton(),
				nfDS.Singleton(),
				connection.ManagerSingleton())
		utils.CrashOnError(err)
	})
	return instance
}
