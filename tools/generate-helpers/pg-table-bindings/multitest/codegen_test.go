// +build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	source := "host=localhost port=5432 database=postgres user=connorgorman sslmode=disable statement_timeout=600000 pool_min_conns=90 pool_max_conns=90"
	config, err := pgxpool.ParseConfig(source)
	if err != nil {
		panic(err)
	}
	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	store := New(pool)

	singleKey := &storage.TestMultiKeyStruct{
		Key1: "key1",
		Key2: "key2",
	}
	dep, exists, err := store.Get(singleKey.GetKey1(), singleKey.GetKey2())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, dep)

	assert.NoError(t, store.Upsert(singleKey))
	dep, exists, err = store.Get(singleKey.GetKey1(), singleKey.GetKey2())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, singleKey, dep)
}
