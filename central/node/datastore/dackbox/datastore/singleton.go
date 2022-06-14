package datastore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/globaldb/dackbox"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/node/datastore/internal/search"
	"github.com/stackrox/stackrox/central/node/datastore/internal/store/postgres"
	"github.com/stackrox/stackrox/central/ranking"
	riskDS "github.com/stackrox/stackrox/central/risk/datastore"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	if features.PostgresDatastore.Enabled() {
		storage := postgres.New(globaldb.GetPostgres(), false)
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
