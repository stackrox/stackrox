package m186tom187

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/m_186_to_m_187_events/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"gorm.io/gorm"
)

var (
	startSeqNum = 186
	migration   = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)},
		Run: func(database *types.Databases) error {
			return createEventsTable(database.PostgresDB, database.GormDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func createEventsTable(_ postgres.DB, gormDB *gorm.DB) error {
	ctx := context.Background()
	pgutils.CreateTableFromModel(ctx, gormDB, frozenSchema.CreateTableEventsStmt)
	return nil
}
