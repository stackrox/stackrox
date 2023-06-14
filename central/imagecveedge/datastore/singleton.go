package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/imagecveedge/datastore/postgres"
	"github.com/stackrox/rox/central/imagecveedge/search"
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
	ad = New(storage, searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
