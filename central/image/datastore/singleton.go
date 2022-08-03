package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/image/datastore/keyfence"
	"github.com/stackrox/rox/central/image/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	if features.PostgresDatastore.Enabled() {
		storage := postgres.New(globaldb.GetPostgres(), keyfence.ImageKeyFenceSingleton())
		indexer := postgres.NewIndexer(globaldb.GetPostgres())
		ad = NewWithPostgres(storage, indexer, riskDS.Singleton(), ranking.ImageRanker(), ranking.ComponentRanker())
		return
	}

	ad = New(dackbox.GetGlobalDackBox(),
		dackbox.GetKeyFence(),
		globalindex.GetGlobalIndex(),
		globalindex.GetProcessIndex(),
		riskDS.Singleton(),
		ranking.ImageRanker(),
		ranking.ComponentRanker())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
