package gatherer

import (
	"github.com/stackrox/rox/central/license/singleton"
	networkEntityDatastore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	gatherer NetworkGraphDefaultExtSrcsGatherer
)

func initialize() {
	gatherer = NewNetworkGraphDefaultExtSrcsGatherer(networkEntityDatastore.Singleton(),
		connection.ManagerSingleton(), singleton.ManagerSingleton())
}

// Singleton returns a singleton instance of NetworkGraphDefaultExtSrcsGatherer.
func Singleton() NetworkGraphDefaultExtSrcsGatherer {
	once.Do(initialize)
	return gatherer
}
