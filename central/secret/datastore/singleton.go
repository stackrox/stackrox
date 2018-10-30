package datastore

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/secret/index"
	"github.com/stackrox/rox/central/secret/search"
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	once sync.Once

	ad DataStore

	logger = logging.LoggerForModule()
)

func initialize() {
	store := store.New(globaldb.GetGlobalDB())
	var err error
	ad, err = New(store, index.New(globalindex.GetGlobalIndex()), search.New(store, globalindex.GetGlobalIndex()))
	if err != nil {
		logger.Panicf("Failed to initialize secrets datastore: %s", err)
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
