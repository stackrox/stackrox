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

var (
	log       = loghelper.LogWrapper{}
	batchSize = 100000
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	tableName := "network_flows_v2"
	column := "updatedat"
	if err := addColumnToTable(ctx, database.PostgresDB, tableName, column); err != nil {
		log.WriteToStderrf("unable to alter table %s: %v", tableName, err)
		return err
	}

	return nil
}

type commandResult interface {
	RowsAffected() int64
}

func addColumnToTable(ctx context.Context, db postgres.DB, table, column string) error {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()
	alterTableStmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s TIMESTAMP WITHOUT TIME ZONE;", table, column)

	_, err := db.Exec(ctx, alterTableStmt)
	if err != nil {
		return errors.Wrapf(err, "unable to alter table %s", table)
	}

	var result commandResult
	for result == nil || batchSize == int(result.RowsAffected()) {
		selectNullStmt := fmt.Sprintf("SELECT flow_id FROM %s WHERE %s IS NULL LIMIT %d", table, column, batchSize)
		updateColumnStmt := fmt.Sprintf("UPDATE %s SET %s = now()::timestamp WHERE flow_id IN (%s);", table, column, selectNullStmt)

		result, err = db.Exec(ctx, updateColumnStmt)
		if err != nil {
			return errors.Wrapf(err, "unable to update column %s", column)
		}
	}

	addIndexStmt := fmt.Sprintf("CREATE INDEX IF NOT EXISTS network_flows_updatedat_v2 ON %s USING brin (%s);", table, column)
	_, err = db.Exec(ctx, addIndexStmt)
	if err != nil {
		return errors.Wrapf(err, "unable to create index in table %s", table)
	}
	return nil
}
