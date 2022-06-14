package datastore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	graphConfigDS "github.com/stackrox/stackrox/central/networkgraph/config/datastore"
	"github.com/stackrox/stackrox/central/networkgraph/entity/datastore/internal/store"
	"github.com/stackrox/stackrox/central/networkgraph/entity/datastore/internal/store/postgres"
	"github.com/stackrox/stackrox/central/networkgraph/entity/datastore/internal/store/rocksdb"
	"github.com/stackrox/stackrox/central/networkgraph/entity/networktree"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once sync.Once
	ds   EntityDataStore
)

// Singleton provides the instance of EntityDataStore to use.
func Singleton() EntityDataStore {
	once.Do(func() {
		var storage store.EntityStore
		var err error
		if features.PostgresDatastore.Enabled() {
			storage = postgres.New(globaldb.GetPostgres())
		} else {
			storage, err = rocksdb.New(globaldb.GetRocksDB())
			utils.CrashOnError(err)
		}
		ds = NewEntityDataStore(storage, graphConfigDS.Singleton(), networktree.Singleton(), connection.ManagerSingleton())
	})
	return ds
}
