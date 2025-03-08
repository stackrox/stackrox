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
	log = loghelper.LogWrapper{}
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
