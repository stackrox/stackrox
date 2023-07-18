package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/image/datastore/keyfence"
	pgStore "github.com/stackrox/rox/central/image/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres(), false, keyfence.ImageKeyFenceSingleton())
	ad = NewWithPostgres(storage, pgStore.NewIndexer(globaldb.GetPostgres()), riskDS.Singleton(), ranking.ImageRanker(), ranking.ComponentRanker())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
