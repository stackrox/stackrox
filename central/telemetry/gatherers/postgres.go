package gatherers

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

type postgresGatherer struct {
	db *pgxpool.Pool
}

func newPostgresGatherer(db *pgxpool.Pool) *postgresGatherer {
	return &postgresGatherer{
		db: db,
	}
}

// Gather returns telemetry information about the Postgres database used by this central
func (d *postgresGatherer) Gather(ctx context.Context) *data.DatabaseStats {
	errorList := errorhelpers.NewErrorList("postgres telemetry gather")
	// Get Postgres config data
	_, adminConfig, err := pgconfig.GetPostgresConfig()
	errorList.AddError(err)

	currentDBBytes, err := pgadmin.GetDatabaseSize(adminConfig, migrations.GetCurrentClone())
	errorList.AddError(err)

	tableStats := globaldb.CollectPostgresStats(ctx, globaldb.GetPostgres())

	dbStats := &data.DatabaseStats{
		Type:      "postgres",
		UsedBytes: currentDBBytes,
		Tables:    tableStats,
		Errors:    errorList.ErrorStrings(),
	}

	// Check Postgres remaining capacity
	availableDBBytes, err := pgadmin.GetRemainingCapacity(adminConfig)
	errorList.AddError(err)

	// In RDS or BYOBD configurations we may not be able to calculate this.
	if availableDBBytes > 0 {
		dbStats.AvailableBytes = availableDBBytes
	}

	return dbStats
}
