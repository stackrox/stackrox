package gatherers

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
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
	errorList := errorhelpers.NewErrorList("postgres telemetry gather")

	totalSize, err := pgadmin.GetTotalPostgresSize(d.adminConfig)
	errorList.AddError(err)

	dbStats := globaldb.CollectPostgresStats(ctx, d.db)
	dbStats.Type = "postgres"
	dbStats.UsedBytes = totalSize
	dbStats.DatabaseDetails = globaldb.CollectPostgresDatabaseSizes(d.adminConfig)
	dbStats.Errors = errorList.ErrorStrings()

	// Check Postgres remaining capacity
	if !env.ManagedCentral.BooleanSetting() {
		availableDBBytes, err := pgadmin.GetRemainingCapacity(d.adminConfig)
		errorList.AddError(err)
		dbStats.AvailableBytes = availableDBBytes
	}

	return dbStats
}
