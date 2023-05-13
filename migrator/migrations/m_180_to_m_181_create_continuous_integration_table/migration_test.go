//go:build sql_integration

package m180tom181

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_180_to_m_181_create_continuous_integration_table/continuousintegrationstore"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())

	pgTest := pghelper.ForT(t, false)

	require.NotNil(t, pgTest)

	store := continuousintegrationstore.New(pgTest.DB)

	// 1. Should not work to get from the non-existing table.
	_, _, err := store.Get(ctx, "some-id")
	assert.ErrorContains(t, err, `relation "continuous_integration_configs" does not exist`)

	// 2. Run the migration and create the table.
	assert.NoError(t, createContinuousIntegrationTable(pgTest.DB, pgTest.GetGormDB()))

	// 3. Test upsert.
	config := &storage.ContinuousIntegrationConfig{
		Id:   uuid.NewV4().String(),
		Type: storage.ContinuousIntegrationType_GITHUB_ACTIONS,
	}

	assert.NoError(t, store.Upsert(ctx, config))

	// 4. Test get.
	returnedConfig, found, err := store.Get(ctx, config.GetId())
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, config, returnedConfig)

	// 5. Test delete.
	assert.NoError(t, store.Delete(ctx, config.GetId()), err)

	// 6. Test that get should not return anything for the ID anymore.
	returnedConfig, found, err = store.Get(ctx, config.GetId())
	assert.NoError(t, err)
	assert.False(t, found)
	assert.Nil(t, returnedConfig)
}
