//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	source := pgtest.GetConnectionString(t)
	config, err := postgres.ParseConfig(source)
	require.NoError(t, err)
	pool, err := postgres.New(context.Background(), config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	Destroy(ctx, pool)
	gormDB := pgtest.OpenGormDB(t, source)
	defer pgtest.CloseGormDB(t, gormDB)
	store := CreateTableAndNewStore(ctx, pool, gormDB)

	multiKey := &storage.TestStruct{
		Key1: "key1",
		Key2: "key2",
	}
	dep, exists, err := store.Get(ctx, multiKey.GetKey1())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, dep)

	assert.NoError(t, store.Upsert(ctx, multiKey))
	dep, exists, err = store.Get(ctx, multiKey.GetKey1())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, multiKey, dep)

	// Search is currently unsupported for tables with multiple primary keys
}
