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

func TestIndex(t *testing.T) {
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
	assert.NoError(t, store.Upsert(ctx, testStruct))

	cases := []struct {
		name         string
		queryBuilder *search.QueryBuilder
		numResults   int
	}{
		{
			name:         "basic string match",
			queryBuilder: search.NewQueryBuilder().AddExactMatches(search.TestKey, testStruct.GetKey()),
			numResults:   1,
		},
		{
			name:         "basic string no match",
			queryBuilder: search.NewQueryBuilder().AddExactMatches(search.TestKey, "nomatch"),
			numResults:   0,
		},

		// TODO
		//{
		//	name: "basic string slice match",
		//	queryBuilder: search.NewQueryBuilder().AddExactMatches(search.TestStringSlice, testStruct.StringSlice[1]),
		//	numResults: 1,
		//},
		//{
		//	name: "basic string slice no match",
		//	queryBuilder: search.NewQueryBuilder().AddExactMatches(search.TestStringSlice, "nomatch"),
		//	numResults: 0,
		//},

		// TODO
		//{
		//	name: "basic bool match",
		//	queryBuilder: search.NewQueryBuilder().AddExactMatches(search.TestStringSlice, testStruct.StringSlice[1]),
		//	numResults: 1,
		//},
		//{
		//	name: "basic bool no match",
		//	queryBuilder: search.NewQueryBuilder().AddExactMatches(search.TestStringSlice, "nomatch"),
		//	numResults: 0,
		//},

		// TODO
		//{
		//	name: "basic bool match",
		//	queryBuilder: search.NewQueryBuilder().AddExactMatches(search.TestStringSlice, testStruct.StringSlice[1]),
		//	numResults: 1,
		//},
		//{
		//	name: "basic bool no match",
		//	queryBuilder: search.NewQueryBuilder().AddExactMatches(search.TestStringSlice, "nomatch"),
		//	numResults: 0,
		//},
	}
	index := NewIndexer(pool)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			results, err := index.Search(c.queryBuilder.ProtoQuery())
			assert.NoError(t, err)
			assert.Len(t, results, c.numResults)
		})
	}
}
