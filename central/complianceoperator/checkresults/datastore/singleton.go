package datastore

import (
	store "github.com/stackrox/rox/central/complianceoperator/checkresults/store"
	pgStore "github.com/stackrox/rox/central/complianceoperator/checkresults/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	ds   DataStore
)

// Singleton returns the singleton datastore
func Singleton() DataStore {
	once.Do(func() {
		var storage store.Store
		storage = pgStore.New(globaldb.GetPostgres())
		ds = NewDatastore(storage)
	})
	return ds
}
