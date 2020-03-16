package pruning

import (
	alertDatastore "github.com/stackrox/rox/central/alert/datastore"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	imagesDatastore "github.com/stackrox/rox/central/image/datastore"
	imageComponentDatastore "github.com/stackrox/rox/central/imagecomponent/datastore"
	networkFlowsDataStore "github.com/stackrox/rox/central/networkflow/datastore"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
	processDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	processWhitelistDatastore "github.com/stackrox/rox/central/processwhitelist/datastore"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	gc   GarbageCollector
)

// Singleton returns the global instance of the garbage collection
func Singleton() GarbageCollector {
	once.Do(func() {
		gc = newGarbageCollector(alertDatastore.Singleton(),
			imagesDatastore.Singleton(),
			clusterDatastore.Singleton(),
			deploymentDatastore.Singleton(),
			podDatastore.Singleton(),
			processDatastore.Singleton(),
			processWhitelistDatastore.Singleton(),
			networkFlowsDataStore.Singleton(),
			configDatastore.Singleton(),
			imageComponentDatastore.Singleton(),
			riskDataStore.Singleton())
	})
	return gc
}
