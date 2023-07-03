package datastore

import (
	"context"
	"testing"

	pgEventsStore "github.com/stackrox/rox/central/events/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
)

func TestWalk(t *testing.T) {
	pool := pgtest.ForT(t)

	store := pgEventsStore.CreateTableAndNewStore(context.Background(), pool.DB, pool.GetGormDB(t))

	ds := New(store)

	events, err := ds.GetEvents(sac.WithAllAccess(context.Background()))
	assert.NoError(t, err)
	assert.Empty(t, events)
}
