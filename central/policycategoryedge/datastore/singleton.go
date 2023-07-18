package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/policycategoryedge/search"
	pgStore "github.com/stackrox/rox/central/policycategoryedge/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	ds = New(storage, search.New(storage, pgStore.NewIndexer(globaldb.GetPostgres())))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
