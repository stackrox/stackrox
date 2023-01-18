package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	dataStore DataStore
)

func initialize() {
	plopStore := postgres.NewFullStore(globaldb.GetPostgres())
	indicatorDataStore := processIndicatorDataStore.Singleton()
	dataStore = New(plopStore, indicatorDataStore)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		log.Warnf("Tried to get datastore singletone for PLOP without Postgres")
		return nil
	}

	once.Do(initialize)
	return dataStore
}
