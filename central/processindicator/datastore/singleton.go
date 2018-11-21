package datastore

import (
	"sync"
	"time"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/pruner"
	"github.com/stackrox/rox/central/processindicator/search"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	pruneInterval     = 10 * time.Minute
	minArgsPerProcess = 5
)

var (
	once sync.Once

	ad DataStore

	log = logging.LoggerForModule()
)

func initialize() {
	storage := store.New(globaldb.GetGlobalDB())
	indexer := index.New(globalindex.GetGlobalIndex())

	searcher, err := search.New(storage, indexer)
	if err != nil {
		panic("unable to load search index for alerts")
	}

	p := pruner.New(minArgsPerProcess, pruneInterval)

	ad = New(storage, indexer, searcher, p)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
