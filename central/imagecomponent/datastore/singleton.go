package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/imagecomponent/datastore/store/postgres"
	"github.com/stackrox/rox/central/imagecomponent/search"
	"github.com/stackrox/rox/central/ranking"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	searcher := search.NewV2(storage, pgStore.NewIndexer(globaldb.GetPostgres()))
	ad = New(storage, searcher, riskDataStore.Singleton(), ranking.ComponentRanker())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
