package manager

import (
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/networkbaseline/datastore"
	networkEntityDS "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	nfDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	networkPolicyDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
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
				connection.ManagerSingleton(),
				networktree.Singleton(),
			)
		utils.CrashOnError(err)
	})
	return instance
}
