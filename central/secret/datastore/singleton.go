package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/secret/internal/index"
	"github.com/stackrox/rox/central/secret/internal/store"
	"github.com/stackrox/rox/central/secret/search"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore

	log = logging.LoggerForModule()
)

func initialize() {
	store := store.New(globaldb.GetGlobalDB())
	indexer := index.New(globalindex.GetGlobalIndex())
	var err error
	ad, err = New(store, indexer, search.New(store, indexer))
	if err != nil {
		log.Panicf("Failed to initialize secrets datastore: %s", err)
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
