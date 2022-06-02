package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	globaldbDackbox "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/imagecomponentedge/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/imagecomponentedge/index"
	"github.com/stackrox/rox/central/imagecomponentedge/search"
	"github.com/stackrox/rox/central/imagecomponentedge/store/dackbox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	if features.PostgresDatastore.Enabled() {
		var err error
		storage := postgres.New(globaldb.GetPostgres())
		indexer := postgres.NewIndexer(globaldb.GetPostgres())
		searcher := search.NewV2(storage, indexer)
		ad, err = New(nil, storage, indexer, searcher)
		utils.CrashOnError(err)
	} else {
		storage, err := dackbox.New(globaldbDackbox.GetGlobalDackBox())
		utils.CrashOnError(err)
		indexer := index.New(globalindex.GetGlobalIndex())
		searcher := search.New(storage, index.New(globalindex.GetGlobalIndex()))

		ad, err = New(globaldbDackbox.GetGlobalDackBox(), storage, indexer, searcher)
		utils.CrashOnError(err)
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
