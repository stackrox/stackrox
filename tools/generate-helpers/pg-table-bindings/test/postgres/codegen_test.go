//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	singleKey = fixtures.GetTestSingleKeyStruct()
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

	testStruct := singleKey.Clone()
	dep, exists, err := store.Get(ctx, testStruct.GetKey())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, dep)

	assert.NoError(t, store.Upsert(ctx, testStruct))
	dep, exists, err = store.Get(ctx, testStruct.GetKey())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, testStruct, dep)
}
