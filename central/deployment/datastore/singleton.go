package datastore

import (
	"sync"

	"github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/deployment/search"
	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	processDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	once sync.Once

	ad DataStore

	logger = logging.LoggerForModule()
)

func initialize() {
	indexer := index.New(globalindex.GetGlobalIndex())

	storage, err := store.New(globaldb.GetGlobalDB())
	if err != nil {
		logger.Panicf("Failed to initialize deployment store: %s", err)
	}

	searcher, err := search.New(storage, indexer)
	if err != nil {
		logger.Panicf("Failed to load deployment index %s", err)
	}

	ad = New(storage, indexer, searcher, processDataStore.Singleton())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
