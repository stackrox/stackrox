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
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sync/semaphore"
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

	var clusters []string
	if err := db.Model(&updatedSchema.Clusters{}).Pluck("id", &clusters).Error; err != nil {
		return err
	}
	log.Infof("clusters found: %v", clusters)

	semaphoreWeight := min(len(clusters), 10)
	var wg sync.WaitGroup
	sema := semaphore.NewWeighted(int64(semaphoreWeight))

	for _, cluster := range clusters {
		if err := sema.Acquire(database.DBCtx, 1); err != nil {
			log.Errorf("context cancelled via stop: %v", err)
			return err
		}

		log.Debugf("Migrate process indicators for cluster %q", cluster)
		wg.Add(1)
		var errorList []error

		go func(c string) {
			defer sema.Release(1)
			defer wg.Done()
			err := migrateByCluster(cluster, database)
			if err != nil {
				errorList = append(errorList, err)
			}
		}(cluster)
		if len(errorList) > 0 {
			return errorList[0]
		}
	}
	wg.Wait()

	return nil
}

func migrateByCluster(cluster string, database *types.Databases) error {
	ctx, cancel := context.WithTimeout(database.DBCtx, types.DefaultMigrationTimeout)
	defer cancel()

	store := updatedStore.New(database.PostgresDB)

	var storeIndicators []*storage.ProcessIndicator
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, cluster).ProtoQuery()
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
