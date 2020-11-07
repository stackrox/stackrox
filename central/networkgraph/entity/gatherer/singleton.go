package gatherer

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/license/singleton"
	networkEntityDatastore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	gatherer NetworkGraphDefaultExtSrcsGatherer
)

func initialize() {
	var err error
	gatherer, err = NewNetworkGraphDefaultExtSrcsGatherer(networkEntityDatastore.Singleton(), singleton.ManagerSingleton())
	utils.Should(errors.Wrap(err, "starting default external sources gatherer"))
}

// Singleton returns a singleton instance of NetworkGraphDefaultExtSrcsGatherer.
func Singleton() NetworkGraphDefaultExtSrcsGatherer {
	once.Do(initialize)
	return gatherer
}
