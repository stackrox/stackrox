package datastore

import (
	pgStore "github.com/stackrox/rox/central/componentcveedge/datastore/store/postgres"
	"github.com/stackrox/rox/central/componentcveedge/search"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	indexer := pgStore.NewIndexer(globaldb.GetPostgres())
	searcher := search.NewV2(storage, indexer)
	ad = New(nil, storage, indexer, searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
