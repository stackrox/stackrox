package m222tom223

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/schema"
)

const deleteComponentRisksStmt = "DELETE FROM " + schema.RisksTableName + " WHERE subject_type = $1 OR subject_type = $2"

func migrate(database *types.Databases) error {
	_, err := database.PostgresDB.Exec(
		database.DBCtx,
		deleteComponentRisksStmt,
		storage.RiskSubjectType_IMAGE_COMPONENT,
		storage.RiskSubjectType_NODE_COMPONENT,
	)
	return err
}
