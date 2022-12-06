package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	store DataStore
)

func initialize() {
	plopStore := postgres.New(globaldb.GetPostgres())
	indicatorDataStore := processIndicatorDataStore.Singleton()
	store = New(plopStore, indicatorDataStore)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return store
}
