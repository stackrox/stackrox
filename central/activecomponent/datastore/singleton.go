package datastore

import (
	acIndexer "github.com/stackrox/rox/central/activecomponent/datastore/index"
	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store/dackbox"
	"github.com/stackrox/rox/central/activecomponent/datastore/search"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	deploymentIndexer "github.com/stackrox/rox/central/deployment/index"
	globaldb "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	componentIndexer "github.com/stackrox/rox/central/imagecomponent/index"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := dackbox.New(globaldb.GetGlobalDackBox(), globaldb.GetKeyFence())
	indexer := acIndexer.New(globalindex.GetGlobalIndex())
	searcher := search.New(storage, globaldb.GetGlobalDackBox(),
		indexer,
		cveIndexer.New(globalindex.GetGlobalIndex()),
		componentIndexer.New(globalindex.GetGlobalIndex()),
		deploymentIndexer.New(globalindex.GetGlobalIndex(), globalindex.GetProcessIndex()))
	ds = New(globaldb.GetGlobalDackBox(), storage, indexer, searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
