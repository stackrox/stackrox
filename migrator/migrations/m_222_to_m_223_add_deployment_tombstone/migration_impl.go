package m222tom223

import (
	"context"

	"github.com/stackrox/rox/migrator/migrations/m_222_to_m_223_add_deployment_tombstone/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	// Use GORM auto-migrate to add the two tombstone columns to the deployments table.
	// No data backfill is needed; NULL means the deployment is active.
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableDeploymentsStmt)
	return nil
}
