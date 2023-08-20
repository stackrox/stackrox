package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/notifications/datastore/internal/search"
	pgStore "github.com/stackrox/rox/central/notifications/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/notifications/datastore/internal/writer"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds  DataStore
	log = logging.LoggerForModule()
)

// Singleton returns a datastore instance to handle notifications.
func Singleton() DataStore {
	if !features.CentralNotifications.Enabled() {
		return nil
	}
	once.Do(func() {
		log.Info("Created the singleton for the datastore")
		searcher := search.New(pgStore.NewIndexer(globaldb.GetPostgres()))
		store := pgStore.New(globaldb.GetPostgres())
		writer := writer.New(store)
		ds = newDataStore(searcher, store, writer)
	})
	return ds
}
