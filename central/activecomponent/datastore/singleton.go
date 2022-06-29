package datastore

import (
	acIndexer "github.com/stackrox/rox/central/activecomponent/datastore/index"
	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store/dackbox"
	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/activecomponent/datastore/search"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	deploymentIndexer "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globaldb"
	globaldbDackbox "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	componentIndexer "github.com/stackrox/rox/central/imagecomponent/index"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	if features.PostgresDatastore.Enabled() {
		storage := postgres.New(globaldb.GetPostgres())
		indexer := postgres.NewIndexer(globaldb.GetPostgres())
		searcher := search.NewV2(storage, indexer)
		ds = New(nil, storage, indexer, searcher)
		return
	}
	storage := dackbox.New(globaldbDackbox.GetGlobalDackBox(), globaldbDackbox.GetKeyFence())
	indexer := acIndexer.New(globalindex.GetGlobalIndex())
	searcher := search.New(storage, globaldbDackbox.GetGlobalDackBox(),
		indexer,
		cveIndexer.New(globalindex.GetGlobalIndex()),
		componentIndexer.New(globalindex.GetGlobalIndex()),
		deploymentIndexer.New(globalindex.GetGlobalIndex(), globalindex.GetProcessIndex()))

	ds = New(globaldbDackbox.GetGlobalDackBox(), storage, indexer, searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
