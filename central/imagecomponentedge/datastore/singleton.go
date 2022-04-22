package datastore

import (
	globaldbDackbox "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/imagecomponentedge/index"
	"github.com/stackrox/rox/central/imagecomponentedge/search"
	"github.com/stackrox/rox/central/imagecomponentedge/store"
	"github.com/stackrox/rox/central/imagecomponentedge/store/dackbox"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var storage store.Store
	var indexer index.Indexer
	var searcher search.Searcher

	// TODO: Wire up.
	// if features.PostgresDatastore.Enabled() {
	//	storage = postgres.New(context.TODO(), globaldb.GetPostgres())
	//	indexer = postgres.NewIndexer(globaldb.GetPostgres())
	//	searcher = search.NewV2(storage, indexer)
	//}

	storage, err := dackbox.New(globaldbDackbox.GetGlobalDackBox())
	utils.CrashOnError(err)
	indexer = index.New(globalindex.GetGlobalIndex())
	searcher = search.New(storage, index.New(globalindex.GetGlobalIndex()))

	ad, err = New(globaldbDackbox.GetGlobalDackBox(), storage, indexer, searcher)
	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
