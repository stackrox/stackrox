//go:build sql_integration
// +build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	source := pgtest.GetConnectionString(t)
	config, err := pgxpool.ParseConfig(source)
	if err != nil {
		panic(err)
	}
	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	Destroy(pool)
	store := New(pool)

	indicator := fixtures.GetProcessIndicator()
	foundIndicator, exists, err := store.Get(indicator.GetId())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, foundIndicator)

	assert.NoError(t, store.Upsert(indicator))
	foundIndicator, exists, err = store.Get(indicator.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, indicator, foundIndicator)

	indicator.ContainerName = "testContainer"
	assert.NoError(t, store.Upsert(indicator))

	foundIndicator, exists, err = store.Get(indicator.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, indicator, foundIndicator)

	assert.NoError(t, store.Delete(indicator.GetId()))
	foundIndicator, exists, err = store.Get(indicator.GetId())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, foundIndicator)

	indicator.ContainerName = "testContainer2"
	assert.NoError(t, store.Upsert(indicator))

	indexer := NewIndexer(pool)

	// Common process indicator searches
	results, err := indexer.Search(search.NewQueryBuilder().AddExactMatches(search.DeploymentID, indicator.DeploymentId).ProtoQuery())
	assert.NoError(t, err)
	assert.Len(t, results, 1)

	// search that finds nothing
	results, err = indexer.Search(search.NewQueryBuilder().AddExactMatches(search.DeploymentID, "71").ProtoQuery())
	assert.NoError(t, err)
	assert.Len(t, results, 0)

	q := search.NewQueryBuilder().
		AddExactMatches(search.DeploymentID, indicator.DeploymentId).
		AddExactMatches(search.ContainerName, indicator.ContainerName).
		ProtoQuery()
	results, err = indexer.Search(q)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}
