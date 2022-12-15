package gatherers

import (
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	depDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/grpc/metrics"
	installation "github.com/stackrox/rox/central/installation/store"
	namespaceDatastore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDatastore "github.com/stackrox/rox/central/node/globaldatastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	sensorUpgradeConfigDatastore "github.com/stackrox/rox/central/sensorupgradeconfig/datastore"
	"github.com/stackrox/rox/pkg/env"
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
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			gatherer = newRoxGatherer(
				newCentralGatherer(
					installation.Singleton(),
					newDatabaseGatherer(
						nil,
						nil,
						nil,
						newPostgresGatherer(globaldb.GetPostgres()),
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
		} else {
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
						nil,
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
		}
	})
	return gatherer
}
