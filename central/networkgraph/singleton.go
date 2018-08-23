package networkgraph

import (
	"sync"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/central/networkpolicies/store"
	networkPolicyStore "github.com/stackrox/rox/central/networkpolicies/store"
)

var (
	once sync.Once

	storage store.Store
	ge      *evaluatorImpl
)

func initialize() {
	ge = newGraphEvaluator(deploymentDataStore.Singleton(),
		namespaceDataStore.New(globaldb.GetGlobalDB()), networkPolicyStore.Singleton())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() Evaluator {
	once.Do(initialize)
	return ge
}
