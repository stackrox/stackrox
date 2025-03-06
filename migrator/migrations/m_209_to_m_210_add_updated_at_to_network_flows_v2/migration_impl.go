package m209tom210

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
)

// TODO(dont-merge): generate/write and import any store required for the migration (skip any unnecessary step):
//  - create a schema subdirectory
//  - create a schema/old subdirectory
//  - create a schema/new subdirectory
//  - create a stores subdirectory
//  - create a stores/previous subdirectory
//  - create a stores/updated subdirectory
//  - copy the old schemas from pkg/postgres/schema to schema/old
//  - copy the old stores from their location in central to appropriate subdirectories in stores/previous
//  - generate the new schemas in pkg/postgres/schema and the new stores where they belong
//  - copy the newly generated schemas from pkg/postgres/schema to schema/new
//  - remove the calls to GetSchemaForTable and to RegisterTable from the copied schema files
//  - remove the xxxTableName constant from the copied schema files
//  - copy the newly generated stores from their location in central to appropriate subdirectories in stores/updated
//  - remove any unused function from the copied store files (the minimum for the public API should contain Walk, UpsertMany, DeleteMany)
//  - remove the scoped access control code from the copied store files
//  - remove the metrics collection code from the copied store files

// TODO(dont-merge): Determine if this change breaks a previous releases database.
// If so increment the `MinimumSupportedDBVersionSeqNum` to the `CurrentDBVersionSeqNum` of the release immediately
// following the release that cannot tolerate the change in pkg/migrations/internal/fallback_seq_num.go.
//
// For example, in 4.2 a column `column_v2` is added to replace the `column_v1` column in 4.1.
// All the code from 4.2 onward will not reference `column_v1`. At some point in the future a rollback to 4.1
// will not longer be supported and we want to remove `column_v1`. To do so, we will upgrade the schema to remove
// the column and update the `MinimumSupportedDBVersionSeqNum` to be the value of `CurrentDBVersionSeqNum` in 4.2
// as 4.1 will no longer be supported. The migration process will inform the user of an error when trying to migrate
// to a software version that can no longer be supported by the database.
var (
	log = loghelper.LogWrapper{}
)

func migrate(database *types.Databases) error {
	_ = database // TODO(dont-merge): remove this line, it is there to make the compiler happy while the migration code is being written.
	// Use databases.DBCtx to take advantage of the transaction wrapping present in the migration initiator

	// TODO(dont-merge): Migration code comes here
	// TODO(dont-merge): When using gorm, make sure you use a separate handle for the updates and the query.  Such as:
	// TODO(dont-merge): db = db.WithContext(database.DBCtx).Table(schema.ListeningEndpointsTableName)
	// TODO(dont-merge): query := db.WithContext(database.DBCtx).Table(schema.ListeningEndpointsTableName).Select("serialized")
	// TODO(dont-merge): See README for more details

	ctx := sac.WithAllAccess(context.Background())
	tableName := "network_flows_v2"
	column := "updatedat"
	if err := addColumnToTable(ctx, database.PostgresDB, tableName, column); err != nil {
		log.WriteToStderrf("unable to alter table %s: %v", tableName, err)
		return err
	}

	//clusters, err := getClusters(ctx, database.PostgresDB)
	//if err != nil {
	//	log.WriteToStderrf("unable to retrieve clusters from network_flows, %v", err)
	//	return err
	//}
	//
	//for _, cluster := range clusters {
	//	tableName := fmt.Sprintf("%s%s", "", cluster)
	//	if err := addColumnToTable(ctx, database.PostgresDB, tableName, "updatedat"); err != nil {
	//		log.WriteToStderrf("unable to alter table %s: %v", tableName, err)
	//		return err
	//	}
	//}

	return nil
}

func addColumnToTable(ctx context.Context, db postgres.DB, table, column string) error {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()
	alterTableStmt := fmt.Sprintf("alter table %s add column if not exists %s timestamp without time zone;", table, column)

	_, err := db.Exec(ctx, alterTableStmt)
	if err != nil {
		return errors.Wrapf(err, "unable to alter table %s", table)
	}

	alterColumnStmt := fmt.Sprintf("alter table %s alter column %s type timestamp without time zone using now()::timestamp;", table, column)

	_, err = db.Exec(ctx, alterColumnStmt)
	if err != nil {
		return errors.Wrapf(err, "unable to alter column %s", column)
	}

	addIndexStmt := fmt.Sprintf("create index if not exists network_flows_updatedat_v2 on %s using brin (%s);", table, column)
	_, err = db.Exec(ctx, addIndexStmt)
	if err != nil {
		return errors.Wrapf(err, "unable to create index in table %s", table)
	}
	return nil
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

// TODO(dont-merge): Write the additional code to support the migration

// TODO(dont-merge): remove any pending TODO
