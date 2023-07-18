package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/serviceaccount/internal/store/postgres"
	"github.com/stackrox/rox/central/serviceaccount/search"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore

	log = logging.LoggerForModule()
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())

	var err error
	ds, err = New(storage, search.New(storage, pgStore.NewIndexer(globaldb.GetPostgres())))
	if err != nil {
		log.Panicf("Failed to initialize secrets datastore: %s", err)
	}
}

// Singleton returns a singleton instance of the service account datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
