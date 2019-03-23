package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/serviceaccount/index"
	"github.com/stackrox/rox/central/serviceaccount/search"
	"github.com/stackrox/rox/central/serviceaccount/store"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore

	logger = logging.LoggerForModule()
)

func initialize() {
	store := store.New(globaldb.GetGlobalDB())
	var err error
	ds, err = New(store, index.New(globalindex.GetGlobalIndex()), search.New(store, globalindex.GetGlobalIndex()))
	if err != nil {
		logger.Panicf("Failed to initialize secrets datastore: %s", err)
	}
}

// Singleton returns a singleton instance of the service account datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
