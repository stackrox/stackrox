package datastore

import (
	globaldb "github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/nodecomponentedge/search"
	pgStore "github.com/stackrox/rox/central/nodecomponentedge/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	searcher := search.New(storage, pgStore.NewIndexer(globaldb.GetPostgres()))
	ad = New(storage, searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
