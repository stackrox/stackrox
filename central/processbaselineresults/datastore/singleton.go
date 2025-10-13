package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/processbaselineresults/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	singleton DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	singleton = New(storage)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return singleton
}
