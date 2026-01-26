package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/image/datastore/keyfence"
	pgStoreV2 "github.com/stackrox/rox/central/image/datastore/store/v2/postgres"
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
	storage := pgStoreV2.New(globaldb.GetPostgres(), false, keyfence.ImageKeyFenceSingleton())
	ad = NewWithPostgres(storage, riskDS.Singleton(), ranking.ImageRanker(), ranking.ComponentRanker())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	if !features.FlattenImageData.Enabled() {
		return nil
	}
	once.Do(initialize)
	return ad
}
