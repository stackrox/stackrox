package m178tom179

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	newStore "github.com/stackrox/rox/migrator/migrations/m_178_to_m_179_embedded_collections_search_label/reportconfigstore"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/m_178_to_m_179_embedded_collections_search_label/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"gorm.io/gorm"
)

const (
	startSeqNum = 178

	batchSize = 500
)

var (
	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)}, // 179
		Run: func(database *types.Databases) error {
			return migrateReportConfigs(database.PostgresDB, database.GormDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateReportConfigs(postgresDB postgres.DB, gormDB *gorm.DB) error {
	ctx := context.Background()
	pgutils.CreateTableFromModel(ctx, gormDB, frozenSchema.CreateTableReportConfigurationsStmt)
	newReportStore := newStore.New(postgresDB)

	reportConfigsToUpsert := make([]*storage.ReportConfiguration, 0, batchSize)
	err := newReportStore.Walk(ctx, func(obj *storage.ReportConfiguration) error {
		reportConfigsToUpsert = append(reportConfigsToUpsert, obj)
		if len(reportConfigsToUpsert) >= batchSize {
			upsertErr := newReportStore.UpsertMany(ctx, reportConfigsToUpsert)
			if upsertErr != nil {
				return upsertErr
			}
			reportConfigsToUpsert = reportConfigsToUpsert[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(reportConfigsToUpsert) > 0 {
		return newReportStore.UpsertMany(ctx, reportConfigsToUpsert)
	}
	return nil
}
