package datastore

import (
	"github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/deployment/search"
	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	nfDS "github.com/stackrox/rox/central/networkflow/datastore"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	pwDS "github.com/stackrox/rox/central/processwhitelist/datastore"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore

	log = logging.LoggerForModule()
)

func initialize() {
	indexer := index.New(globalindex.GetGlobalIndex())

	storage, err := store.New(globaldb.GetGlobalDB())
	if err != nil {
		log.Panicf("Failed to initialize deployment store: %s", err)
	}

	searcher, err := search.New(storage, indexer)
	if err != nil {
		log.Panicf("Failed to load deployment index %s", err)
	}

	ad = New(storage, indexer, searcher, piDS.Singleton(), pwDS.Singleton(), nfDS.Singleton())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
