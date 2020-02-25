package datastore

import (
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/processwhitelist/index"
	"github.com/stackrox/rox/central/processwhitelist/search"
	"github.com/stackrox/rox/central/processwhitelist/store"
	"github.com/stackrox/rox/central/processwhitelistresults/datastore"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore

	log = logging.LoggerForModule()
)

func initialize() {
	storage := store.Singleton()
	indexer := index.New(globalindex.GetGlobalTmpIndex())

	searcher, err := search.New(storage, indexer)
	if err != nil {
		panic("unable to load search index for process whitelist")
	}

	ad = New(storage, indexer, searcher, datastore.Singleton())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
