package gatherer

import (
	blobstore "github.com/stackrox/rox/central/blob/datastore"
	networkEntityDatastore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	gatherer NetworkGraphDefaultExtSrcsGatherer
)

func initialize() {
	gatherer = NewNetworkGraphDefaultExtSrcsGatherer(networkEntityDatastore.Singleton(), blobstore.Singleton())
}

// Singleton returns a singleton instance of NetworkGraphDefaultExtSrcsGatherer.
func Singleton() NetworkGraphDefaultExtSrcsGatherer {
	once.Do(initialize)
	return gatherer
}
