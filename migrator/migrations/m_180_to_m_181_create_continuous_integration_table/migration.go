package m_180_to_m_181_create_continuous_integration_table

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/m_180_to_m_181_create_continuous_integration_table/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"gorm.io/gorm"
)

var (
	startSeqNum = 180
	migration   = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)},
		Run: func(databases *types.Databases) error {
			return createContinuousIntegrationTable(databases.PostgresDB, databases.GormDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func createContinuousIntegrationTable(_ postgres.DB, gormDB *gorm.DB) error {
	pgutils.CreateTableFromModel(context.Background(), gormDB, frozenSchema.CreateTableContinuousIntegrationConfigsStmt)
	return nil
}
