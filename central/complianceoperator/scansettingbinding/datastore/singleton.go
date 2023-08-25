package datastore

import (
	pgStore "github.com/stackrox/rox/central/complianceoperator/scansettingbinding/store/postgres"
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
		storage := pgStore.New(globaldb.GetPostgres())
		ds = NewDatastore(storage)
	})
	return ds
}
