package m225tom226

import (
	"context"

	"github.com/stackrox/rox/migrator/migrations/m_225_to_m_226_bg_add_deployment_type_and_enforcement_count_to_alerts/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableAlertsStmt)
	return nil
}
