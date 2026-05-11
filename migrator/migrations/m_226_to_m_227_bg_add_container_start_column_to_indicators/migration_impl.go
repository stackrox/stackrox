package m226tom227

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/sac"
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()

	_, err := database.PostgresDB.Exec(ctx,
		fmt.Sprintf("ALTER TABLE process_indicators ADD COLUMN IF NOT EXISTS %s TIMESTAMP", "bg_containerstarttime"))
	if err != nil {
		return errors.Wrap(err, "adding bg_containerstarttime column to process_indicators")
	}
	return nil
}
