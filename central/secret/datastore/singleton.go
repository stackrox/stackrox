package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/secret/internal/store/postgres"
	"github.com/stackrox/rox/central/secret/search"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	ad = New(storage, search.New(storage))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
