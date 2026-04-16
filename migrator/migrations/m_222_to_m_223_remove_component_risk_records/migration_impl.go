package m222tom223

import (
	"github.com/stackrox/rox/generated/storage"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/m_222_to_m_223_remove_component_risk_records/schema"
	"github.com/stackrox/rox/migrator/types"
)

const deleteComponentRisksStmt = "DELETE FROM " + frozenSchema.RisksTableName + " WHERE subject_type = $1 OR subject_type = $2"

func migrate(database *types.Databases) error {
	_, err := database.PostgresDB.Exec(
		database.DBCtx,
		deleteComponentRisksStmt,
		storage.RiskSubjectType_IMAGE_COMPONENT,
		storage.RiskSubjectType_NODE_COMPONENT,
	)
	return err
}
