package m175tom176

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/m_183_to_m_184_create_declarative_config_health_table/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"gorm.io/gorm"
)

var (
	startSeqNum = 183
	migration   = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)},
		Run: func(database *types.Databases) error {
			return createDeclarativeConfigHealthTable(database.PostgresDB, database.GormDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func createDeclarativeConfigHealthTable(_ postgres.DB, gormDB *gorm.DB) error {
	ctx := context.Background()
	pgutils.CreateTableFromModel(ctx, gormDB, schema.CreateTableDeclarativeConfigHealthsStmt)
	return nil
}
