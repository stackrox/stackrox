package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/fixtures"
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
	fmt.Println(store)

	deployment := fixtures.GetDeployment()
	dep, exists, err := store.Get(deployment.GetId())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, dep)

	assert.NoError(t, store.Upsert(deployment))
	dep, exists, err = store.Get(deployment.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, deployment, dep)

	deployment.Name = "newname"
	assert.NoError(t, store.Upsert(deployment))

	dep, exists, err = store.Get(deployment.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, deployment, dep)

	err = store.Delete(deployment.GetId())
	dep, exists, err = store.Get(deployment.GetId())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, dep)
}
