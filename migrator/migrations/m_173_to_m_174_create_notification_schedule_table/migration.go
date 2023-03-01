package m173tom174

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v75"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"gorm.io/gorm"
)

var (
	startSeqNum = 173
	migration   = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)},
		Run: func(database *types.Databases) error {
			// Migration code comes here
			return createNotificationScheduleTable(database.PostgresDB, database.GormDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

// Additional code to support the migration

func createNotificationScheduleTable(_ *postgres.DB, gormDB *gorm.DB) error {
	ctx := context.Background()
	pgutils.CreateTableFromModel(ctx, gormDB, frozenSchema.CreateTableNotificationSchedulesStmt)
	return nil
}
