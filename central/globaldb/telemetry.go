package globaldb

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// Gather database related information.
// Current properties we gather:
// "Database size"
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)
	props := make(map[string]any)

	_, config, err := pgconfig.GetPostgresConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get postgres config")
	}

	dbSize, err := pgadmin.GetDatabaseSize(config, pgconfig.GetActiveDB())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databaze size")
	}

	_ = phonehome.AddTotal(ctx, props, "Database Size", func(_ context.Context) (int, error) {
		return int(dbSize), nil
	})

	return props, nil
}
