package gatherers

import (
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	depDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/grpc/metrics"
	installation "github.com/stackrox/rox/central/installation/store"
	manager "github.com/stackrox/rox/central/license/singleton"
	namespaceDatastore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDatastore "github.com/stackrox/rox/central/node/globaldatastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/gatherers"
)

var (
	gatherer     *RoxGatherer
	gathererInit sync.Once
)

// Singleton initializes and returns a RoxGatherer singleton
func Singleton() *RoxGatherer {
	gathererInit.Do(func() {
		gatherer = newRoxGatherer(
			newCentralGatherer(
				manager.ManagerSingleton(),
				installation.Singleton(),
				newDatabaseGatherer(newBadgerGatherer(globaldb.GetGlobalBadgerDB()), newBoltGatherer(globaldb.GetGlobalDB()), newBleveGatherer(globalindex.GetGlobalIndex())),
				newAPIGatherer(metrics.GRPCSingleton(), metrics.HTTPSingleton()),
				gatherers.NewComponentInfoGatherer(),
			),
			newClusterGatherer(
				clusterDatastore.Singleton(),
				nodeDatastore.Singleton(),
				namespaceDatastore.Singleton(),
				connection.ManagerSingleton(),
				depDatastore.Singleton(),
			),
		)
	})
	return gatherer
}
