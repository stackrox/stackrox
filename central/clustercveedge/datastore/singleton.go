package datastore

import (
	pgStore "github.com/stackrox/rox/central/clustercveedge/datastore/store/postgres"
	"github.com/stackrox/rox/central/clustercveedge/index"
	"github.com/stackrox/rox/central/clustercveedge/search"
	"github.com/stackrox/rox/central/clustercveedge/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var err error
	var storage store.Store
	var indexer index.Indexer

	storage = pgStore.NewFullStore(globaldb.GetPostgres())
	indexer = pgStore.NewIndexer(globaldb.GetPostgres())
	ad, err = New(nil, storage, indexer, search.NewV2(storage, indexer))
	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
