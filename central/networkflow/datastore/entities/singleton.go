package entities

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/networkflow/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once
	ds   EntityDataStore
)

// Singleton provides the instance of EntityDataStore to use.
func Singleton() EntityDataStore {
	storage, err := rocksdb.New(globaldb.GetRocksDB())
	utils.Must(err)

	once.Do(func() {
		ds = NewEntityDataStore(storage)
	})
	return ds
}
