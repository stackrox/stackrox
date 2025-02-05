package globaldb

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// Gather database related information.
// Current properties we gather:
// "Database size"
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	props := make(map[string]any)

	_, config, err := pgconfig.GetPostgresConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get postgres config")
	}

	dbSize, err := pgadmin.GetDatabaseSize(config, pgconfig.GetActiveDB())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databaze size")
	}

	props["Database Size"] = dbSize

	return props, nil
}
