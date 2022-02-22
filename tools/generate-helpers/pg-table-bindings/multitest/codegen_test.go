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

	store := New(pool)

	multiKey := &storage.TestMultiKeyStruct{
		Key1: "key1",
		Key2: "key2",
	}
	dep, exists, err := store.Get(multiKey.GetKey1(), multiKey.GetKey2())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, dep)

	assert.NoError(t, store.Upsert(multiKey))
	dep, exists, err = store.Get(multiKey.GetKey1(), multiKey.GetKey2())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, multiKey, dep)
}
