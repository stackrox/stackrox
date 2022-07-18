package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/policycategory/index"
	"github.com/stackrox/rox/central/policycategory/search"
	policyCategoryStore "github.com/stackrox/rox/central/policycategory/store"
	policyCategoryPostgres "github.com/stackrox/rox/central/policycategory/store/postgres"
	"github.com/stackrox/rox/central/policycategory/store/rocksdb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var storage policyCategoryStore.Store
	var indexer index.Indexer

	if features.PostgresDatastore.Enabled() {
		storage = policyCategoryPostgres.New(globaldb.GetPostgres())
		indexer = policyCategoryPostgres.NewIndexer(globaldb.GetPostgres())
	} else {
		storage = rocksdb.New(globaldb.GetRocksDB())
		indexer = index.New(globalindex.GetGlobalTmpIndex())
	}
	addDefaults(storage)
	searcher := search.New(storage, indexer)

	ad = New(storage, indexer, searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}

// TODO: implement addDefaults adds the default categories into the postgres table for policy categories.
func addDefaults(s policyCategoryStore.Store) {

}
