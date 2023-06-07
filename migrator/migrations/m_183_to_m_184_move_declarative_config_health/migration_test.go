//go:build sql_integration

package m183tom184

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_183_to_m_184_move_declarative_config_health/declarativeconfig/store"
	integrationHealthSchema "github.com/stackrox/rox/migrator/migrations/m_183_to_m_184_move_declarative_config_health/integrationhealth/schema"
	integrationHealthStore "github.com/stackrox/rox/migrator/migrations/m_183_to_m_184_move_declarative_config_health/integrationhealth/store"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	pgTest := pghelper.ForT(t, false)
	require.NotNil(t, pgTest)
	pgutils.CreateTableFromModel(ctx, pgTest.GetGormDB(), integrationHealthSchema.CreateTableIntegrationHealthsStmt)

	integrationHealthStore := integrationHealthStore.New(pgTest.DB)

	unhealthyDeclarativeConfigName := "Club-Mate Zero"
	unhealthyDeclarativeConfigID := uuid.NewV5FromNonUUIDs("role", unhealthyDeclarativeConfigName).String()
	unhealthyDeclarativeConfig := &storage.IntegrationHealth{
		Id:           unhealthyDeclarativeConfigID,
		Name:         unhealthyDeclarativeConfigName,
		Status:       storage.IntegrationHealth_UNHEALTHY,
		ErrorMessage: "not enough sugar",
		Type:         storage.IntegrationHealth_DECLARATIVE_CONFIG,
	}
	err := integrationHealthStore.Upsert(ctx, unhealthyDeclarativeConfig)
	assert.NoError(t, err)

	healthyDeclarativeConfigName := "Club-Mate Original"
	healthyDeclarativeConfigID := uuid.NewV5FromNonUUIDs("role", healthyDeclarativeConfigName).String()
	healthyDeclarativeConfig := &storage.IntegrationHealth{
		Id:     healthyDeclarativeConfigID,
		Name:   healthyDeclarativeConfigName,
		Status: storage.IntegrationHealth_HEALTHY,
		Type:   storage.IntegrationHealth_DECLARATIVE_CONFIG,
	}
	err = integrationHealthStore.Upsert(ctx, healthyDeclarativeConfig)
	assert.NoError(t, err)

	healthyBackupName := "Club-Mate Extra"
	healthyBackupID := uuid.NewV4().String()
	healthyBackupHealth := &storage.IntegrationHealth{
		Id:     healthyBackupID,
		Name:   healthyBackupName,
		Status: storage.IntegrationHealth_HEALTHY,
		Type:   storage.IntegrationHealth_BACKUP,
	}
	err = integrationHealthStore.Upsert(ctx, healthyBackupHealth)
	assert.NoError(t, err)

	healthStore := store.New(pgTest.DB)
	// Test walk in the not-created-table returns an error.
	errPre := healthStore.Walk(ctx, func(obj *storage.DeclarativeConfigHealth) error { return nil })
	assert.ErrorContains(t, errPre, "relation \"declarative_config_healths\" does not exist")

	// Run migration.
	assert.NoError(t, moveDeclarativeConfigHealthToNewStore(pgTest.DB, pgTest.GetGormDB()))

	totalSize := 0
	err = healthStore.Walk(ctx, func(obj *storage.DeclarativeConfigHealth) error {
		totalSize++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, totalSize)

	config, exists, err := healthStore.Get(ctx, unhealthyDeclarativeConfigID)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, unhealthyDeclarativeConfigName, config.GetName())
	assert.Equal(t, unhealthyDeclarativeConfig.GetErrorMessage(), config.GetErrorMessage())
	assert.Equal(t, unhealthyDeclarativeConfig.GetStatus().String(), config.GetStatus().String())
	assert.Equal(t, unhealthyDeclarativeConfig.GetLastTimestamp(), config.GetLastTimestamp())

	config, exists, err = healthStore.Get(ctx, healthyDeclarativeConfigID)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, healthyDeclarativeConfigName, config.GetName())
	assert.Equal(t, healthyDeclarativeConfig.GetErrorMessage(), config.GetErrorMessage())
	assert.Equal(t, healthyDeclarativeConfig.GetStatus().String(), config.GetStatus().String())
	assert.Equal(t, healthyDeclarativeConfig.GetLastTimestamp(), config.GetLastTimestamp())
}
