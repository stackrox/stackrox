package gatherers

import (
	"context"
	"errors"
	"strings"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

type postgresGatherer struct {
	db          postgres.DB
	adminConfig *postgres.Config
}

func newPostgresGatherer(db postgres.DB, adminConfig *postgres.Config) *postgresGatherer {
	return &postgresGatherer{
		db:          db,
		adminConfig: adminConfig,
	}
}

// Gather returns telemetry information about the Postgres database used by this central
func (d *postgresGatherer) Gather(ctx context.Context) *data.DatabaseStats {
	var gathererErrs error
	dbStats := globaldb.CollectPostgresStats(ctx, d.db)
	dbStats.Type = "postgres"

	dbStats.DatabaseDetails = globaldb.CollectPostgresDatabaseSizes(d.adminConfig)

	// Check Postgres remaining capacity
	if !env.ManagedCentral.BooleanSetting() && !pgconfig.IsExternalDatabase() {
		totalSize, err := pgadmin.GetTotalPostgresSize(d.adminConfig)
		if err != nil {
			gathererErrs = errors.Join(gathererErrs, err)
		}
		dbStats.UsedBytes = totalSize

		availableDBBytes, err := pgadmin.GetRemainingCapacity(d.adminConfig)
		if err != nil {
			gathererErrs = errors.Join(gathererErrs, err)
		}
		dbStats.AvailableBytes = availableDBBytes
	}

	if gathererErrs != nil {
		dbStats.Errors = strings.Split(gathererErrs.Error(), "\n")
	}

	return dbStats
}
