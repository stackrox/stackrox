package datastore

import (
	"github.com/stackrox/stackrox/central/activecomponent/datastore/internal/store/dackbox"
	"github.com/stackrox/stackrox/central/activecomponent/datastore/search"
	acIndexer "github.com/stackrox/stackrox/central/activecomponent/index"
	cveIndexer "github.com/stackrox/stackrox/central/cve/index"
	deploymentIndexer "github.com/stackrox/stackrox/central/deployment/index"
	globaldb "github.com/stackrox/stackrox/central/globaldb/dackbox"
	"github.com/stackrox/stackrox/central/globalindex"
	componentIndexer "github.com/stackrox/stackrox/central/imagecomponent/index"
	"github.com/stackrox/stackrox/pkg/sync"
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
