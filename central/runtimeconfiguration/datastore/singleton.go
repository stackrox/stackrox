package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/runtimeconfiguration/store/postgres"
	pgCollectionsStore "github.com/stackrox/rox/central/runtimeconfigurationcollection/store/postgres"

	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	dataStore DataStore
)

func initialize() {
	store := pgStore.New(globaldb.GetPostgres())
	collectionsStore := pgCollectionsStore.New(globaldb.GetPostgres())
	pool := globaldb.GetPostgres()
	dataStore = New(store, collectionsStore, pool)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return dataStore
}
