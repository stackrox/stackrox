package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	graphConfigDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	pgStore "github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	ds   EntityDataStore
)

// Singleton provides the instance of EntityDataStore to use.
func Singleton() EntityDataStore {
	once.Do(func() {
		storage := pgStore.New(globaldb.GetPostgres())
		ds = NewEntityDataStore(storage, graphConfigDS.Singleton(), networktree.Singleton(), connection.ManagerSingleton())
	})
	return ds
}
