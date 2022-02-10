package pruning

import (
	alertDatastore "github.com/stackrox/rox/central/alert/datastore"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	imagesDatastore "github.com/stackrox/rox/central/image/datastore"
	imageComponentDatastore "github.com/stackrox/rox/central/imagecomponent/datastore"
	networkFlowsDataStore "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	nodeGlobalDatastore "github.com/stackrox/rox/central/node/globaldatastore"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
	processBaselineDatastore "github.com/stackrox/rox/central/processbaseline/datastore"
	processDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	vulnReqDataStore "github.com/stackrox/rox/central/vulnerabilityrequest/datastore"
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
			nodeGlobalDatastore.Singleton(),
			imagesDatastore.Singleton(),
			clusterDatastore.Singleton(),
			deploymentDatastore.Singleton(),
			podDatastore.Singleton(),
			processDatastore.Singleton(),
			processBaselineDatastore.Singleton(),
			networkFlowsDataStore.Singleton(),
			configDatastore.Singleton(),
			imageComponentDatastore.Singleton(),
			riskDataStore.Singleton(),
			vulnReqDataStore.Singleton())
	})
	return gc
}
