package m223tom224

import (
	"context"

	"github.com/stackrox/rox/migrator/migrations/m_224_to_m_225_set_deployment_state/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	// Ensure the state and deleted columns exist before backfilling.
	// The migrator applies schema changes automatically, but the migration
	// sequence number may run before the schema is applied.
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableDeploymentsStmt)

	// Backfill existing rows: the new state column defaults to NULL in
	// PostgreSQL, but all pre-existing deployments are active. Set
	// state = 0 (DEPLOYMENT_STATE_ACTIVE) for any row that was not
	// explicitly set yet.
	_, err := database.PostgresDB.Exec(ctx,
		"UPDATE deployments SET state = 0 WHERE state IS NULL")
	return err
}
