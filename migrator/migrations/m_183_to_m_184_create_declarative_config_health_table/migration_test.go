//go:build sql_integration

package m175tom176

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_183_to_m_184_create_declarative_config_health_table/store"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	pgTest := pghelper.ForT(t, false)
	require.NotNil(t, pgTest)

	healthStore := store.New(pgTest.DB)
	// Test walk in the not-created-table returns an error
	errPre := healthStore.Walk(ctx, func(obj *storage.DeclarativeConfigHealth) error { return nil })
	assert.ErrorContains(t, errPre, "relation \"declarative_config_healths\" does not exist")

	// Create the table
	assert.NoError(t, createDeclarativeConfigHealthTable(pgTest.DB, pgTest.GetGormDB()))

	// Test walk in created table returns nil
	errPost := healthStore.Walk(ctx, func(obj *storage.DeclarativeConfigHealth) error {
		return nil
	})
	assert.NoError(t, errPost)

	// Test upsert is possible
	declarativeConfigHealth := &storage.DeclarativeConfigHealth{
		Id:   uuid.New().String(),
		Name: "Club-Mate Zero",
	}
	upsertErr := healthStore.Upsert(ctx, declarativeConfigHealth)
	assert.NoError(t, upsertErr)

	// Test get now retrieves the upserted item
	fetchedHealth, found, fetchErr := healthStore.Get(ctx, declarativeConfigHealth.GetId())
	assert.NoError(t, fetchErr)
	assert.True(t, found)
	assert.Equal(t, declarativeConfigHealth, fetchedHealth)

	// Test Delete
	deleteErr := healthStore.Delete(ctx, fetchedHealth.GetId())
	assert.NoError(t, deleteErr)

	deletedHealth, deletedFound, deletedRetrieveErr := healthStore.Get(ctx, declarativeConfigHealth.GetId())
	assert.NoError(t, deletedRetrieveErr)
	assert.False(t, deletedFound)
	assert.Nil(t, deletedHealth)
}
