package singleton

import (
	"github.com/stackrox/rox/central/clusterinit/store"
	pgStore "github.com/stackrox/rox/central/clusterinit/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	instance     store.Store
	instanceInit sync.Once
)

// Singleton returns the singleton data store for cluster init bundles.
func Singleton() store.Store {
	instanceInit.Do(func() {
		underlying := pgStore.New(globaldb.GetPostgres())
		instance = store.NewStore(underlying)
	})
	return instance
}
