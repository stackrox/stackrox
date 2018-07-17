package networkgraph

import (
	"sync"

	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	globaldb "bitbucket.org/stack-rox/apollo/central/globaldb/singletons"
	namespaceDataStore "bitbucket.org/stack-rox/apollo/central/namespace/store"
	"bitbucket.org/stack-rox/apollo/central/networkpolicies/store"
	networkPolicyStore "bitbucket.org/stack-rox/apollo/central/networkpolicies/store"
)

var (
	once sync.Once

	storage store.Store
	ge      *graphEvaluatorImpl
)

func initialize() {
	ge = newGraphEvaluator(clusterDataStore.Singleton(), deploymentDataStore.Singleton(),
		namespaceDataStore.New(globaldb.GetGlobalDB()), networkPolicyStore.Singleton())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() GraphEvaluator {
	once.Do(initialize)
	return ge
}
