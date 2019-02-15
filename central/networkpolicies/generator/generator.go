package generator

import (
	"github.com/stackrox/rox/central/deployment/datastore"
	flowStore "github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/central/networkpolicies/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Generator encapsulates the logic of the network policy generator.
type Generator interface {
	Generate(req *v1.GenerateNetworkPoliciesRequest) (generated []*storage.NetworkPolicy, toDelete []*v1.NetworkPolicyReference, err error)
}

// New creates and returns a new network policy generator.
func New(networkPolicyStore store.Store, deploymentStore datastore.DataStore, globalFlowStore flowStore.ClusterStore) Generator {
	return &generator{
		networkPolicyStore: networkPolicyStore,
		deploymentStore:    deploymentStore,
		globalFlowStore:    globalFlowStore,
	}
}
