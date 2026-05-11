package m002tom003

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/backgroundmigrations/migrations"
	"github.com/stackrox/rox/central/backgroundmigrations/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	log       = logging.LoggerForModule()
	batchSize = 100000
)

func init() {
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum:     2,
		VersionAfterSeqNum: 3,
		Description:        "Backfill bg_updatedat column in network_flows_v2",
		Run:                run,
	})
}

func run(ctx context.Context, db postgres.DB) error {
	table := "network_flows_v2"
	column := "bg_updatedat"

	totalUpdated := int64(0)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		selectNullStmt := fmt.Sprintf("SELECT flow_id FROM %s WHERE %s IS NULL LIMIT %d", table, column, batchSize)
		updateStmt := fmt.Sprintf("UPDATE %s SET %s = now()::timestamp WHERE flow_id IN (%s)", table, column, selectNullStmt)

		result, err := db.Exec(ctx, updateStmt)
		if err != nil {
			return errors.Wrapf(err, "updating column %s", column)
		}

		affected := result.RowsAffected()
		totalUpdated += affected
		log.Infof("Backfilled bg_updatedat for %d network flows (total: %d)", affected, totalUpdated)

		if affected < int64(batchSize) {
			break
		}
	}

	addIndexStmt := fmt.Sprintf("CREATE INDEX IF NOT EXISTS network_flows_bg_updatedat_v2 ON %s USING brin (%s)", table, column)
	if _, err := db.Exec(ctx, addIndexStmt); err != nil {
		return errors.Wrapf(err, "creating index on %s.%s", table, column)
	}

	log.Infof("Successfully backfilled bg_updatedat for %d total network flows", totalUpdated)
	return nil
}
