package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/image/datastore/store/postgres"
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
		ad = NewWithPostgres(storage, indexer, riskDS.Singleton(), ranking.ImageRanker(), ranking.ComponentRanker())
		return
	}

	ad = New(dackbox.GetGlobalDackBox(),
		dackbox.GetKeyFence(),
		globalindex.GetGlobalIndex(),
		globalindex.GetProcessIndex(),
		false,
		riskDS.Singleton(),
		ranking.ImageRanker(),
		ranking.ComponentRanker())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}

// ImageKeyFenceSingleton provides a key fence for image and its sub-components.
func ImageKeyFenceSingleton() concurrency.KeyFence {
	once.Do(initialize)
	return kf
}
