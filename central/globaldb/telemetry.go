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
//
// "Database size (bytes)" - Total size of the active database in PostgreSQL
// cluster in bytes.
//
// "PostgreSQL version" - Text representation of PostgreSQL version in form
// "X.Y", where X is the major version and Y is the minor version. Might
// include non-numeric parts as well.
//
// "Database is external" - Whether the PostgreSQL installation we use is the
// ACS default that comes out of the box, or is it an external database
// provided by the user.
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

	db := GetPostgres()
	version := GetPostgresVersion(ctx, db)

	props["Database size (bytes)"] = dbSize
	props["PostgreSQL version"] = version
	props["Database is external"] = pgconfig.IsExternalDatabase()

	return props, nil
}
