package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	pgStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	dataStore DataStore
)

func initialize() {
	plopStore := pgStore.NewFullStore(globaldb.GetPostgres())
	indicatorDataStore := processIndicatorDataStore.Singleton()
	dataStore = New(plopStore, indicatorDataStore)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return dataStore
}
