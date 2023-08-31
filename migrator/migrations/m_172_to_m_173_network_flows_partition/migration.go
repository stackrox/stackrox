package m172tom173

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_172_to_m_173_network_flows_partition/schema"
	"github.com/stackrox/rox/migrator/migrations/m_172_to_m_173_network_flows_partition/stores/previous"
	"github.com/stackrox/rox/migrator/migrations/m_172_to_m_173_network_flows_partition/stores/updated"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"gorm.io/gorm"
)

var (
	startSeqNum = 172

	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)}, // 173
		Run: func(databases *types.Databases) error {
			err := MigrateToPartitions(databases.GormDB, databases.PostgresDB)
			if err != nil {
				return errors.Wrap(err, "updating network_flows to partitions")
			}
			return nil
		},
	}

	log = loghelper.LogWrapper{}
)

// MigrateToPartitions updates the btree network flow indexes to be hash
func MigrateToPartitions(gormDB *gorm.DB, db postgres.DB) error {
	ctx := sac.WithAllAccess(context.Background())

	err := analyzeOldTable(ctx, db)
	if err != nil {
		log.WriteToStderrf("unable to analyze network_flows.  Will continue processing though it may be slow. %v", err)
	}

	// First get the distinct clusters in the network_flows table
	clusters, err := getClusters(ctx, db)
	if err != nil {
		log.WriteToStderrf("unable to retrieve clusters from network_flows, %v", err)
		return err
	}

	// Now apply the updated schema to create a partition table with updated index types.  The
	// individual partitions will be created on a per cluster basis as the store is created.
	pgutils.CreateTableFromModel(ctx, gormDB, updatedSchema.CreateTableNetworkFlowsStmt)

	// Create the partition and move the data
	for _, cluster := range clusters {
		sourceStore := previous.New(db, cluster)

		previousCount, err := getPreviousCount(ctx, sourceStore)
		if err != nil {
			return err
		}
		log.WriteToStderrf("Found %d total network flows to consider for migration.", previousCount)

		// Create the updated store which will create the partition
		destinationStore := updated.New(db, cluster)

		err = migrateData(ctx, db, destinationStore.GetPartitionName(), cluster)
		if err != nil {
			log.WriteToStderrf("unable to move data for cluster %q, %v", cluster, err)
			return err
		}

		migratedCount, err := getDestinationCount(ctx, destinationStore)
		if err != nil {
			return err
		}

		if migratedCount != previousCount {
			return errors.Wrapf(err, "Expected to migrate %d records but only migrated %d records for cluster %q. ", previousCount, migratedCount, cluster)
		}

		// Ideally this would have been done on the source.  However, the reason we are implementing
		// this change is because removing the stale flows was becoming problematic with large amounts of data.
		// So we will copy it all over and then remove the stale data from the partition once it is migrated.
		err = cleanupDestinationPartition(ctx, destinationStore)
		if err != nil {
			return err
		}

		migratedCount, err = getDestinationCount(ctx, destinationStore)
		if err != nil {
			return err
		}
		log.WriteToStderrf("Trimmed network flows to length of %d from %d.", migratedCount, previousCount)
	}

	// Drop the old table
	err = gormDB.Migrator().DropTable("network_flows")
	if err != nil {
		log.WriteToStderrf("unable to drop table network_flows, %v", err)
		return err
	}

	return nil
}

func getPreviousCount(ctx context.Context, store previous.FlowStore) (int, error) {
	// Due to heavy load with network flows, each call to the store could take a long time so we will
	// pass a context with the migration timeout to them.
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()

	return store.Count(ctx)
}

func getDestinationCount(ctx context.Context, store updated.FlowStore) (int, error) {
	// Due to heavy load with network flows, each call to the store could take a long time so we will
	// pass a context with the migration timeout to them.
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()

	return store.Count(ctx)
}

func cleanupDestinationPartition(ctx context.Context, store updated.FlowStore) error {
	// Due to heavy load with network flows, each call to the store could take a long time so we will
	// pass a context with the migration timeout to them.
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()

	return store.RemoveStaleFlows(ctx)
}

func analyzeOldTable(ctx context.Context, db postgres.DB) error {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()

	_, err := db.Exec(ctx, "ANALYZE network_flows;")
	return err
}

func getClusters(ctx context.Context, db postgres.DB) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()

	var clusters []string
	getClustersStmt := "select distinct id from clusters;"

	rows, err := db.Query(ctx, getClustersStmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var cluster string
		if err := rows.Scan(&cluster); err != nil {
			return nil, err
		}

		clusters = append(clusters, cluster)
	}

	return clusters, rows.Err()
}

func migrateData(ctx context.Context, db postgres.DB, partitionName string, cluster string) error {
	clusterUUID, err := uuid.FromString(cluster)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()

	// Skip the serial ID
	moveDataStmt := fmt.Sprintf("INSERT INTO %s (Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, LastSeenTimestamp, ClusterId) SELECT Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, LastSeenTimestamp, ClusterId FROM network_flows WHERE ClusterId = $1", partitionName)

	_, err = db.Exec(ctx, moveDataStmt, clusterUUID)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
