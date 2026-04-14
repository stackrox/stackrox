package m223tom224

import (
	"context"

	"github.com/stackrox/rox/migrator/migrations/m_223_to_m_224_add_deleted_at_index_and_set_deployment_state/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
)

const (
	addIndexStmt       = "CREATE INDEX IF NOT EXISTS deployments_deleted ON deployments (deleted)"
	setActiveStateStmt = "UPDATE deployments SET state = 1 WHERE state IS NULL OR state = 0"
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	// Add deleted and state columns if they do not already exist.
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableDeploymentsStmt)

	// Add an index on deleted for efficient soft-delete queries.
	if _, err := database.PostgresDB.Exec(database.DBCtx, addIndexStmt); err != nil {
		return err
	}

	// Set all existing deployments with STATE_UNSPECIFIED (0) to STATE_ACTIVE (1).
	if _, err := database.PostgresDB.Exec(database.DBCtx, setActiveStateStmt); err != nil {
		return err
	}

	return nil
}
