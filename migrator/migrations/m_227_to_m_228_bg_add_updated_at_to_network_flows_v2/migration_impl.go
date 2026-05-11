package m227tom228

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
		fmt.Sprintf("ALTER TABLE network_flows_v2 ADD COLUMN IF NOT EXISTS %s TIMESTAMP WITHOUT TIME ZONE", "bg_updatedat"))
	if err != nil {
		return errors.Wrap(err, "adding bg_updatedat column to network_flows_v2")
	}
	return nil
}
