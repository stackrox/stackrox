package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/reports/config/store/postgres"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

var (
	once sync.Once
	ds   DataStore
)

// Singleton creates a singleton for the report configuration datastore and loads the plugin client config
func Singleton() DataStore {
	once.Do(func() {
		storage := pgStore.New(globaldb.GetPostgres())
		ds = New(storage)
	})
	return ds
}
