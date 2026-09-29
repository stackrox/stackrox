package m226tom227

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/migrator/migrations/m_226_to_m_227_backfill_report_type/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	log = loghelper.LogWrapper{}
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableReportSnapshotsStmt)

	result, err := database.PostgresDB.Exec(ctx, `UPDATE report_snapshots SET type = 0 WHERE type IS NULL`)
	if err != nil {
		return errors.Wrap(err, "failed to backfill report_snapshots.type")
	}
	log.WriteToStderrf("Backfilled type=0 for %d report snapshots", result.RowsAffected())
	return nil
}
