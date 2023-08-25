package m187tom188

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/migrations/m_187_to_m_188_remove_reports_without_migrated_collections/schema"
	"github.com/stackrox/rox/migrator/types"
)

// The goal of this migration is to ensure that all reports have a non-empty scopeid that points to a valid collection
// Note that `resourcescope_collectionid is a new field introduced in 4.2 and shouldn't be set at the beginning
// of this migration, so it will not be considered here. In rare cases, where the user upgrades, creates a _new_ report
// with resourcescope_collectionid and then rollsback, then this migration will delete that new data.
// However, since there is no guarantee that this new data is persisted after a rollback, this is acceptable.
func migrate(database *types.Databases) error {
	ctx, cancel := context.WithTimeout(context.Background(), types.DefaultMigrationTimeout)
	defer cancel()

	// Delete all reports whose scopeid is not found in the collections table
	sql := fmt.Sprintf(
		"DELETE FROM %[1]s WHERE NOT EXISTS (SELECT 1 FROM %[2]s WHERE %[1]s.scopeid = %[2]s.id)",
		schema.ReportConfigurationsTableName,
		schema.CollectionsTableName,
	)
	r, err := database.PostgresDB.Exec(ctx, sql)

	if err != nil {
		return errors.Wrap(err, "failed to delete invalid report configurations")
	}

	if r.RowsAffected() != 0 {
		log.Infof("Deleted %d report configurations because they have invalid collections", r.RowsAffected())
	} else {
		log.Debug("No invalid report configurations found")
	}

	return nil
}
