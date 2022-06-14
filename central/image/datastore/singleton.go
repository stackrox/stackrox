package datastore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/globaldb/dackbox"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/image/datastore/internal/store/postgres"
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
