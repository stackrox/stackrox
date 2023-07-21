package gatherers

import (
	"github.com/pkg/errors"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	depDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	installation "github.com/stackrox/rox/central/installation/store"
	namespaceDatastore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	sensorUpgradeConfigDatastore "github.com/stackrox/rox/central/sensorupgradeconfig/datastore"
	"github.com/stackrox/rox/pkg/grpc/metrics"
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
		_, adminConfig, err := pgconfig.GetPostgresConfig()
		utils.CrashOnError(errors.Wrap(err, "unable to get Postgres config"))

		dbGatherer := newDatabaseGatherer(
			newPostgresGatherer(globaldb.GetPostgres(), adminConfig),
		)

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
