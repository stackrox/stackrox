package gatherers

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

type postgresGatherer struct {
	db          *pgxpool.Pool
	adminConfig *pgxpool.Config
}

func newPostgresGatherer(db *pgxpool.Pool, adminConfig *pgxpool.Config) *postgresGatherer {
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
	availableDBBytes, err := pgadmin.GetRemainingCapacity(d.adminConfig)
	errorList.AddError(err)

	// In RDS or BYOBD configurations we may not be able to calculate this.
	if availableDBBytes > 0 {
		dbStats.AvailableBytes = availableDBBytes
	}

	return dbStats
}
