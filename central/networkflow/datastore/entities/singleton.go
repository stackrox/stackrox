package entities

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/networkflow/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once
	ds   EntityDataStore
)

// Singleton provides the instance of EntityDataStore to use.
func Singleton() EntityDataStore {
	if !features.NetworkGraphExternalSrcs.Enabled() {
		utils.Should(errors.Errorf("%s feature is disabled", features.NetworkGraphExternalSrcs.Name()))
	}

	storage, err := rocksdb.New(globaldb.GetRocksDB())
	utils.Must(err)

	once.Do(func() {
		ds = NewEntityDataStore(storage)
	})
	return ds
}
