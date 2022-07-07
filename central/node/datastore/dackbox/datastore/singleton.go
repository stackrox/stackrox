package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/node/datastore/internal/search"
	"github.com/stackrox/rox/central/node/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore

	kf concurrency.KeyFence
)

func initialize() {
	if features.PostgresDatastore.Enabled() {
		kf = concurrency.NewKeyFence()
		storage := postgres.New(globaldb.GetPostgres(), false, kf)
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

// NodeKeyFenceSingleton provides a key fence for node and its sub-components.
func NodeKeyFenceSingleton() concurrency.KeyFence {
	once.Do(initialize)
	return kf
}
