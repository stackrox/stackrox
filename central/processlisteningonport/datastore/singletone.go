package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	store DataStore
)

func initialize() {
	plopStorage := postgres.New(globaldb.GetPostgres())
	indicatorStorage := processIndicatorStore.Singleton()
	store = New(plopStorage, indicatorStorage)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return store
}
