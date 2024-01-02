package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/teams/store"
	pgStore "github.com/stackrox/rox/central/teams/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

type DataStore interface {
	GetTeam(ctx context.Context, id string) (*storage.Team, bool, error)
	ListTeams(ctx context.Context) ([]*storage.Team, error)
	AddTeam(ctx context.Context, team *storage.Team) (*storage.Team, error)
	GetTeamsByName(ctx context.Context, names ...string) ([]*storage.Team, error)
}

func newDatastore(store store.Store) DataStore {
	return &dataStoreImpl{store: store}
}

// Singleton provides a singleton datastore for teams.
func Singleton() DataStore {
	once.Do(func() {
		ds = newDatastore(pgStore.New(globaldb.GetPostgres()))
	})
	return ds
}
