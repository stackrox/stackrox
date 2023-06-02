package m183tom184

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/m_183_to_m_184_move_declarative_config_health/declarativeconfig/schema"
	declarativeConfigStore "github.com/stackrox/rox/migrator/migrations/m_183_to_m_184_move_declarative_config_health/declarativeconfig/store"
	integrationHealthStore "github.com/stackrox/rox/migrator/migrations/m_183_to_m_184_move_declarative_config_health/integrationhealth/store"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/utils"
	"gorm.io/gorm"
)

var (
	startSeqNum = 183
	migration   = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)},
		Run: func(database *types.Databases) error {
			return moveDeclarativeConfigHealthToNewStore(database.PostgresDB, database.GormDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func moveDeclarativeConfigHealthToNewStore(db postgres.DB, gormDB *gorm.DB) error {
	ctx := context.Background()
	pgutils.CreateTableFromModel(ctx, gormDB, schema.CreateTableDeclarativeConfigHealthsStmt)

	configStore := declarativeConfigStore.New(db)
	integrationHealthStore := integrationHealthStore.New(db)
	idsToDelete := make([]string, 0)

	err := integrationHealthStore.Walk(ctx, func(obj *storage.IntegrationHealth) error {
		if obj.GetType() == storage.IntegrationHealth_DECLARATIVE_CONFIG {
			if err := configStore.Upsert(ctx, convertIntegrationHealthToDeclarativeHealth(obj)); err != nil {
				return err
			}
			idsToDelete = append(idsToDelete, obj.GetId())
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, id := range idsToDelete {
		if err := integrationHealthStore.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func convertIntegrationHealthToDeclarativeHealth(obj *storage.IntegrationHealth) *storage.DeclarativeConfigHealth {
	healthStatus := utils.IfThenElse(obj.GetStatus() == storage.IntegrationHealth_UNHEALTHY, storage.DeclarativeConfigHealth_UNHEALTHY, storage.DeclarativeConfigHealth_HEALTHY)
	return &storage.DeclarativeConfigHealth{
		Id:            obj.GetId(),
		Name:          obj.GetName(),
		Status:        healthStatus,
		ErrorMessage:  obj.GetErrorMessage(),
		LastTimestamp: obj.GetLastTimestamp(),
	}
}
