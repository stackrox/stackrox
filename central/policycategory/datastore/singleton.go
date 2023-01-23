package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/policycategory/index"
	"github.com/stackrox/rox/central/policycategory/search"
	policyCategoryStore "github.com/stackrox/rox/central/policycategory/store"
	policyCategoryPostgres "github.com/stackrox/rox/central/policycategory/store/postgres"
	"github.com/stackrox/rox/central/policycategory/store/rocksdb"
	policyCategoryEdgeDS "github.com/stackrox/rox/central/policycategoryedge/datastore"
	"github.com/stackrox/rox/pkg/defaults/categories"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var store policyCategoryStore.Store
	var indexer index.Indexer

	if features.NewPolicyCategories.Enabled() && env.PostgresDatastoreEnabled.BooleanSetting() {
		store = policyCategoryPostgres.New(globaldb.GetPostgres())
		indexer = policyCategoryPostgres.NewIndexer(globaldb.GetPostgres())
	} else {
		store = rocksdb.New(globaldb.GetRocksDB())
		indexer = index.New(globalindex.GetGlobalTmpIndex())
	}
	addDefaults(store)
	searcher := search.New(store, indexer)
	ad = New(store, indexer, searcher, policyCategoryEdgeDS.Singleton())

}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}

func addDefaults(s policyCategoryStore.Store) {
	// Preload the default policies.
	defaultCategories, err := categories.DefaultPolicyCategories()
	// Hard panic here is okay, since we can always guarantee that we will be able to get the default policies out.
	utils.CrashOnError(err)

	for _, p := range defaultCategories {
		if err := s.Upsert(policyCategoryCtx, p); err != nil {
			utils.CrashOnError(err)
		}
	}
	log.Infof("Loaded %d default policy categories", len(defaultCategories))

}
