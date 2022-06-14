package pruning

import (
	alertDatastore "github.com/stackrox/stackrox/central/alert/datastore"
	clusterDatastore "github.com/stackrox/stackrox/central/cluster/datastore"
	configDatastore "github.com/stackrox/stackrox/central/config/datastore"
	deploymentDatastore "github.com/stackrox/stackrox/central/deployment/datastore"
	imagesDatastore "github.com/stackrox/stackrox/central/image/datastore"
	imageComponentDatastore "github.com/stackrox/stackrox/central/imagecomponent/datastore"
	networkFlowsDataStore "github.com/stackrox/stackrox/central/networkgraph/flow/datastore"
	nodeGlobalDatastore "github.com/stackrox/stackrox/central/node/globaldatastore"
	podDatastore "github.com/stackrox/stackrox/central/pod/datastore"
	processBaselineDatastore "github.com/stackrox/stackrox/central/processbaseline/datastore"
	processDatastore "github.com/stackrox/stackrox/central/processindicator/datastore"
	k8sRoleDataStore "github.com/stackrox/stackrox/central/rbac/k8srole/datastore"
	k8srolebindingStore "github.com/stackrox/stackrox/central/rbac/k8srolebinding/datastore"
	riskDataStore "github.com/stackrox/stackrox/central/risk/datastore"
	serviceAccountDataStore "github.com/stackrox/stackrox/central/serviceaccount/datastore"
	vulnReqDataStore "github.com/stackrox/stackrox/central/vulnerabilityrequest/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
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
			vulnReqDataStore.Singleton(),
			serviceAccountDataStore.Singleton(),
			k8sRoleDataStore.Singleton(),
			k8srolebindingStore.Singleton())
	})
	return gc
}
