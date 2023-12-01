package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/risk/datastore/internal/search"
	pgStore "github.com/stackrox/rox/central/risk/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	ad = New(storage, search.New(storage, pgStore.NewIndexer(globaldb.GetPostgres())))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
