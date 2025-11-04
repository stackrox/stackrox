package m212tom213

import (
	"context"
	"slices"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_212_to_m_213_add_container_start_column_to_indicators/schema"
	updatedStore "github.com/stackrox/rox/migrator/migrations/m_212_to_m_213_add_container_start_column_to_indicators/store"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/search"
)

var (
	batchSize = 5000
	log       = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	// We are simply promoting a field to a column so the serialized object is unchanged.  Thus, we
	// have no need to worry about the old schema and can simply perform all our work on the new one.
	db := database.GormDB
	pgutils.CreateTableFromModel(database.DBCtx, db, updatedSchema.CreateTableProcessIndicatorsStmt)
	db = db.WithContext(database.DBCtx).Table(updatedSchema.ClustersTableName)

	// Drop the indexes except the cluster one.
	resultDB := db.Exec("DROP INDEX if exists processindicators_deploymentid")
	if resultDB.Error != nil {
		log.Error(errors.Wrap(resultDB.Error, "unable to drop index processindicators_deploymentid"))
	}
	resultDB = db.Exec("DROP INDEX if exists processindicators_poduid")
	if resultDB.Error != nil {
		log.Error(errors.Wrap(resultDB.Error, "unable to drop index processindicators_poduid"))
	}
	resultDB = db.Exec("DROP INDEX if exists processindicators_signal_time")
	if resultDB.Error != nil {
		log.Error(errors.Wrap(resultDB.Error, "unable to drop index processindicators_signal_time"))
	}

	var clusters []string
	if err := db.Model(&updatedSchema.Clusters{}).Pluck("id", &clusters).Error; err != nil {
		return err
	}
	log.Infof("clusters found: %v", clusters)

	// Run sequentially to avoid pgx concurrent map writes issue
	for _, cluster := range clusters {
		log.Debugf("Migrate process indicators for cluster %q", cluster)
		if err := migrateByCluster(cluster, database); err != nil {
			return err
		}
	}

	// Add the indexes back
	resultDB = db.Exec("CREATE INDEX CONCURRENTLY IF NOT EXISTS processindicators_deploymentid ON process_indicators USING HASH (deploymentid)")
	if resultDB.Error != nil {
		log.Error(errors.Wrap(resultDB.Error, "unable to create index processindicators_deploymentid"))
	}
	resultDB = db.Exec("CREATE INDEX CONCURRENTLY IF NOT EXISTS processindicators_poduid ON process_indicators USING HASH (poduid)")
	if resultDB.Error != nil {
		log.Error(errors.Wrap(resultDB.Error, "unable to create index processindicators_poduid"))
	}
	resultDB = db.Exec("CREATE INDEX CONCURRENTLY IF NOT EXISTS processindicators_signal_time ON process_indicators (signal_time)")
	if resultDB.Error != nil {
		log.Error(errors.Wrap(resultDB.Error, "unable to create index processindicators_signal_time"))
	}

	log.Info("Process Indicators migrated")
	return nil
}

func migrateByCluster(cluster string, database *types.Databases) error {
	ctx, cancel := context.WithTimeout(database.DBCtx, types.DefaultMigrationTimeout)
	defer cancel()

	store := updatedStore.New(database.PostgresDB)

	var storeIndicators []*storage.ProcessIndicator
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, cluster).AddNullField(search.ProcessContainerStartTime).ProtoQuery()
	storeIndicators, err := store.GetByQuery(ctx, query)
	if err != nil {
		return err
	}

	log.Infof("Processing %s with %d indicators", cluster, len(storeIndicators))
	recordsMigrated := 0

	for objBatch := range slices.Chunk(storeIndicators, batchSize) {
		if err = store.UpsertMany(ctx, objBatch); err != nil {
			return errors.Wrap(err, "failed to upsert all converted objects")
		}
		recordsMigrated += len(objBatch)
	}

	log.Infof("Populated container start time for %d process indicators in cluster %s", recordsMigrated, cluster)

	return nil
}
