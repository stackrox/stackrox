package fetcher

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterCVEEdgeDataStore "github.com/stackrox/rox/central/clustercveedge/datastore"
	clusterCVEDS "github.com/stackrox/rox/central/cve/cluster/datastore"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	manager OrchestratorIstioCVEManager
	once    sync.Once
)

// SingletonManager returns a singleton instance of OrchestratorIstioCVEManager
func SingletonManager() OrchestratorIstioCVEManager {
	var err error
	once.Do(func() {
		clusterCVEDatastore := clusterCVEDS.Singleton()
		manager, err = NewOrchestratorIstioCVEManagerImpl(clusterDataStore.Singleton(), clusterCVEDatastore,
			clusterCVEEdgeDataStore.Singleton(), cveMatcher.Singleton())
		utils.CrashOnError(err)
	})

	return manager
}
