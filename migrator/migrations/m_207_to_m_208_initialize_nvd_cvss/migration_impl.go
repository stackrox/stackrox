package m207tom208

import (
	newSchema "github.com/stackrox/rox/migrator/migrations/m_207_to_m_208_initialize_nvd_cvss/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
)

// If so increment the `MinimumSupportedDBVersionSeqNum` to the `CurrentDBVersionSeqNum` of the release immediately
// following the release that cannot tolerate the change in pkg/migrations/internal/fallback_seq_num.go.
//
// For example, in 4.2 a column `column_v2` is added to replace the `column_v1` column in 4.1.
// All the code from 4.2 onward will not reference `column_v1`. At some point in the future a rollback to 4.1
// will not longer be supported and we want to remove `column_v1`. To do so, we will upgrade the schema to remove
// the column and update the `MinimumSupportedDBVersionSeqNum` to be the value of `CurrentDBVersionSeqNum` in 4.2
// as 4.1 will no longer be supported. The migration process will inform the user of an error when trying to migrate
// to a software version that can no longer be supported by the database.

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
