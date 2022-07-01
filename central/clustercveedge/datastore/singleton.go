package datastore

import (
	"github.com/stackrox/rox/central/clustercveedge/datastore/store/postgres"
	"github.com/stackrox/rox/central/clustercveedge/index"
	"github.com/stackrox/rox/central/clustercveedge/search"
	"github.com/stackrox/rox/central/clustercveedge/store"
	"github.com/stackrox/rox/central/clustercveedge/store/dackbox"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	"github.com/stackrox/rox/central/globaldb"
	globalDackBox "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/features"
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

	if features.PostgresDatastore.Enabled() {
		storage = postgres.NewFullStore(globaldb.GetPostgres())
		indexer = postgres.NewIndexer(globaldb.GetPostgres())
		ad, err = New(nil, storage, indexer, search.NewV2(storage, indexer))
		utils.CrashOnError(err)
		return
	}

	storage, err = dackbox.New(globalDackBox.GetGlobalDackBox(), globalDackBox.GetKeyFence())
	utils.CrashOnError(err)

	searcher := search.New(storage, index.New(globalindex.GetGlobalIndex()), cveIndexer.New(globalindex.GetGlobalIndex()), globalDackBox.GetGlobalDackBox())

	ad, err = New(globalDackBox.GetGlobalDackBox(), storage, index.New(globalindex.GetGlobalIndex()), searcher)
	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
