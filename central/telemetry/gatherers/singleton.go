package gatherers

import (
	"github.com/pkg/errors"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	depDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/grpc/metrics"
	installation "github.com/stackrox/rox/central/installation/store"
	namespaceDatastore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	sensorUpgradeConfigDatastore "github.com/stackrox/rox/central/sensorupgradeconfig/datastore"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/gatherers"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	gatherer     *RoxGatherer
	gathererInit sync.Once
)

// Singleton initializes and returns a RoxGatherer singleton
func Singleton() *RoxGatherer {
	gathererInit.Do(func() {
		var dbGatherer *databaseGatherer

		if env.PostgresDatastoreEnabled.BooleanSetting() {
			_, adminConfig, err := pgconfig.GetPostgresConfig()
			utils.CrashOnError(errors.Wrap(err, "unable to get Postgres config"))

			dbGatherer = newDatabaseGatherer(
				nil,
				nil,
				nil,
				newPostgresGatherer(globaldb.GetPostgres(), adminConfig),
			)
		} else {
			dbGatherer = newDatabaseGatherer(
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
			)
		}

		gatherer = newRoxGatherer(
			newCentralGatherer(
				installation.Singleton(),
				dbGatherer,
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
