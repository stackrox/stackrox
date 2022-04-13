package gatherer

import (
	networkEntityDatastore "github.com/stackrox/stackrox/central/networkgraph/entity/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	gatherer NetworkGraphDefaultExtSrcsGatherer
)

func initialize() {
	gatherer = NewNetworkGraphDefaultExtSrcsGatherer(networkEntityDatastore.Singleton())
}

// Singleton returns a singleton instance of NetworkGraphDefaultExtSrcsGatherer.
func Singleton() NetworkGraphDefaultExtSrcsGatherer {
	once.Do(initialize)
	return gatherer
}
