package datastore

import (
	store "github.com/stackrox/stackrox/central/complianceoperator/rules/store"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once sync.Once
	ds   DataStore
)

// Singleton returns the singleton datastore
func Singleton() DataStore {
	once.Do(func() {
		store, err := store.New(globaldb.GetRocksDB())
		utils.CrashOnError(err)

		ds, err = NewDatastore(store)
		utils.CrashOnError(err)
	})
	return ds
}
