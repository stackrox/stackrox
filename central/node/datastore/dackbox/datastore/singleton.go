package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/node/datastore/dackbox/datastore/keyfence"
	"github.com/stackrox/rox/central/node/datastore/search"
	"github.com/stackrox/rox/central/node/datastore/store/postgres"
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
		storage := postgres.New(globaldb.GetPostgres(), false, keyfence.NodeKeyFenceSingleton())
		indexer := postgres.NewIndexer(globaldb.GetPostgres())
		searcher := search.NewV2(storage, indexer)
		ad = NewWithPostgres(storage, indexer, searcher, riskDS.Singleton(), ranking.NodeRanker(), ranking.NodeComponentRanker())
		return
	}

	ad = New(dackbox.GetGlobalDackBox(),
		dackbox.GetKeyFence(),
		globalindex.GetGlobalIndex(),
		riskDS.Singleton(),
		ranking.NodeRanker(),
		ranking.ComponentRanker())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
