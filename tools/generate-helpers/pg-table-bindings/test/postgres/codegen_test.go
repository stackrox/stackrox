// +build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

var (
	singleKey = &storage.TestSingleKeyStruct{
		Key: "key1",
		Name: "name",
		StringSlice: []string {
			"slice1", "slice2",
		},
		Bool: true,
		Uint64: 16,
		Int64: 32,
		Float: 4.56,
		Labels: map[string]string {
			"key1": "value1",
			"key2": "value2",
		},
		Timestamp: &types.Timestamp{
			Seconds:              1645640515,
			Nanos:                0,
		},
		Enum: storage.TestSingleKeyStruct_ENUM1,
		Enums: []storage.TestSingleKeyStruct_Enum {
			storage.TestSingleKeyStruct_ENUM1,
			storage.TestSingleKeyStruct_ENUM2,
		},
	}
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

	testStruct := singleKey.Clone()
	dep, exists, err := store.Get(testStruct.GetKey())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, dep)

	assert.NoError(t, store.Upsert(testStruct))
	dep, exists, err = store.Get(testStruct.GetKey())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, testStruct, dep)
}

func TestIndex(t *testing.T) {
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

	testStruct := singleKey.Clone()
	assert.NoError(t, store.Upsert(testStruct))

	cases := []struct {
		name string
		queryBuilder *search.QueryBuilder
		numResults int
	} {
		{
			name: "basic string match",
			queryBuilder: search.NewQueryBuilder().AddExactMatches(search.TestKey, testStruct.GetKey()),
			numResults: 1,
		},
		{
			name: "basic string no match",
			queryBuilder: search.NewQueryBuilder().AddExactMatches(search.TestKey, "nomatch"),
			numResults: 0,
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

		/*
		singleKey = &storage.TestSingleKeyStruct{
				Key: "key1",
				Name: "name",
				StringSlice: []string {
					"slice1", "slice2",
				},
				Bool: true,
				Uint64: 16,
				Int64: 32,
				Float: 4.56,
				Labels: map[string]string {
					"key1": "value1",
					"key2": "value2",
				},
				Timestamp: &types.Timestamp{
					Seconds:              1645640515,
					Nanos:                0,
				},
				Enum: storage.TestSingleKeyStruct_ENUM1,
				Enums: []storage.TestSingleKeyStruct_Enum {
					storage.TestSingleKeyStruct_ENUM1,
					storage.TestSingleKeyStruct_ENUM2,
				},
			}
		 */
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
