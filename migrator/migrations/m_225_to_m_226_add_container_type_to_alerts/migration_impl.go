package m225tom226

import (
	"github.com/stackrox/rox/migrator/migrations/m_225_to_m_226_add_container_type_to_alerts/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
)

func migrate(database *types.Databases) error {
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, schema.CreateTableAlertsStmt)
	return nil
}
