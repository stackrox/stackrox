package fetcher

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	manager K8sIstioCVEManager
	once    sync.Once
)

// SingletonManager returns a singleton instance of k8sCVEManager
func SingletonManager() K8sIstioCVEManager {
	var err error
	once.Do(func() {
		manager, err = Newk8sIstioCVEManagerImpl(clusterDataStore.Singleton(), cveDataStore.Singleton(), cveMatcher.Singleton())
		utils.Must(err)
	})

	return manager
}
