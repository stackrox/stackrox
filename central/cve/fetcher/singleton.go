package fetcher

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterCVEEdgeDataStore "github.com/stackrox/rox/central/clustercveedge/datastore"
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
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
		// TODO: Replace with cluster CVE datastore
		manager, err = NewOrchestratorIstioCVEManagerImpl(clusterDataStore.Singleton(), cveDataStore.Singleton(), clusterCVEEdgeDataStore.Singleton(), cveMatcher.Singleton())
		utils.CrashOnError(err)
	})

	return manager
}
