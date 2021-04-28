package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/role/datastore/internal/store"
	simpleAccessScopeStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/rocksdb"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	ds   DataStore
	once sync.Once
)

// Singleton returns the singleton providing access to the roles store.
func Singleton() DataStore {
	once.Do(func() {
		accessScopeStorage, err := simpleAccessScopeStore.New(globaldb.GetRocksDB())
		utils.Must(err)

		ds = New(store.Singleton(), accessScopeStorage)
	})
	return ds
}
