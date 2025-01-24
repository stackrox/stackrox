package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/image/datastore/keyfence"
	"github.com/stackrox/rox/central/image/datastore/store"
	pgStore "github.com/stackrox/rox/central/image/datastore/store/postgres"
	v2Store "github.com/stackrox/rox/central/image/datastore/store/v2/postgres"
	componentStore "github.com/stackrox/rox/central/imagecomponent/v2/datastore/store/postgres"
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
	var storage store.Store
	if features.FlattenCVEData.Enabled() {
		componentV2Store := componentStore.New(globaldb.GetPostgres())
		storage = v2Store.New(globaldb.GetPostgres(), false, keyfence.ImageKeyFenceSingleton(), componentV2Store)
	} else {
		storage = pgStore.New(globaldb.GetPostgres(), false, keyfence.ImageKeyFenceSingleton())
	}
	ad = NewWithPostgres(storage, riskDS.Singleton(), ranking.ImageRanker(), ranking.ComponentRanker())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
