package gatherers

import (
	clusterDatastore "github.com/stackrox/stackrox/central/cluster/datastore"
	depDatastore "github.com/stackrox/stackrox/central/deployment/datastore"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/grpc/metrics"
	installation "github.com/stackrox/stackrox/central/installation/store"
	namespaceDatastore "github.com/stackrox/stackrox/central/namespace/datastore"
	nodeDatastore "github.com/stackrox/stackrox/central/node/globaldatastore"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	sensorUpgradeConfigDatastore "github.com/stackrox/stackrox/central/sensorupgradeconfig/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/telemetry/gatherers"
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
				installation.Singleton(),
				newDatabaseGatherer(
					newRocksDBGatherer(globaldb.GetRocksDB()),
					newBoltGatherer(globaldb.GetGlobalDB()),
					newBleveGatherer(
						globalindex.GetGlobalIndex(),
						globalindex.GetGlobalTmpIndex(),
						globalindex.GetAlertIndex(),
						globalindex.GetPodIndex(),
						globalindex.GetProcessIndex(),
					),
				),
				newAPIGatherer(metrics.GRPCSingleton(), metrics.HTTPSingleton()),
				gatherers.NewComponentInfoGatherer(),
				sensorUpgradeConfigDatastore.Singleton(),
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
