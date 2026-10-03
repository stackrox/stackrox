package runner

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/migrator/version"
)

func updateVersion(ctx context.Context, databases *types.Databases, newVersion *storage.Version) error {
	return version.UpdateVersionPostgres(ctx, databases.PostgresDB, newVersion)
}
