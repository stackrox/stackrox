package datastore

import (
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/policy/index"
	"github.com/stackrox/rox/central/policy/search"
	"github.com/stackrox/rox/central/policy/store"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := store.New(globaldb.GetGlobalDB())
	indexer := index.New(globalindex.GetGlobalTmpIndex())
	clusterDatastore := clusterDS.Singleton()
	notiferDatastore := notifierDS.Singleton()

	searcher, err := search.New(storage, indexer)
	if err != nil {
		panic("unable to load search index for alerts")
	}

	ad = New(storage, indexer, searcher, clusterDatastore, notiferDatastore)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
