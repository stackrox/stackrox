package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/image/datastore/keyfence"
	pgStore "github.com/stackrox/rox/central/image/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		storage := pgStore.New(globaldb.GetPostgres(), false, keyfence.ImageKeyFenceSingleton())
		indexer := pgStore.NewIndexer(globaldb.GetPostgres())
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
