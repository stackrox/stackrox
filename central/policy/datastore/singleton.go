package datastore

import (
	clusterDS "github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/globalindex"
	notifierDS "github.com/stackrox/stackrox/central/notifier/datastore"
	"github.com/stackrox/stackrox/central/policy/index"
	"github.com/stackrox/stackrox/central/policy/search"
	"github.com/stackrox/stackrox/central/policy/store"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := store.New(globaldb.GetGlobalDB())
	indexer := index.New(globalindex.GetGlobalTmpIndex())
	searcher := search.New(storage, indexer)

	clusterDatastore := clusterDS.Singleton()
	notiferDatastore := notifierDS.Singleton()

	ad = New(storage, indexer, searcher, clusterDatastore, notiferDatastore)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
