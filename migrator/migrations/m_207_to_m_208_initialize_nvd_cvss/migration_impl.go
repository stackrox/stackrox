package m207tom208

import (
	newSchema "github.com/stackrox/rox/migrator/migrations/m_207_to_m_208_initialize_nvd_cvss/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
)

func migrate(database *types.Databases) error {
	// Use databases.DBCtx to take advantage of the transaction wrapping present in the migration initiator
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, newSchema.CreateTableImageCvesStmt)

	db := database.PostgresDB

	updateStmt := `update image_cves set nvdcvss = 0 where nvdcvss is NULL`
	_, err := db.Exec(database.DBCtx, updateStmt)
	if err != nil {
		return err
	}

	return nil
}
