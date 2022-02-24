// +build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
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

	singleKey := &storage.TestSingleKeyStruct{
		Key: "key1",
	}
	dep, exists, err := store.Get(singleKey.GetKey())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, dep)

	assert.NoError(t, store.Upsert(singleKey))
	dep, exists, err = store.Get(singleKey.GetKey())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, singleKey, dep)
}
