//go:build sql_integration
// +build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	singleKey = fixtures.GetTestSingleKeyStruct()
)

func TestStore(t *testing.T) {
	ctx := context.Background()

	source := pgtest.GetConnectionString(t)
	config, err := pgxpool.ParseConfig(source)
	require.NoError(t, err)
	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	Destroy(ctx, pool)
	store := New(ctx, pool)

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
