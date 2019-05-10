package generator

import (
	"context"

	dDS "github.com/stackrox/rox/central/deployment/datastore"
	nsDS "github.com/stackrox/rox/central/namespace/datastore"
	flowStore "github.com/stackrox/rox/central/networkflow/store"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Generator encapsulates the logic of the network policy generator.
type Generator interface {
	Generate(ctx context.Context, req *v1.GenerateNetworkPoliciesRequest) (generated []*storage.NetworkPolicy, toDelete []*storage.NetworkPolicyReference, err error)
}

// New creates and returns a new network policy generator.
func New(networkPolicies npDS.DataStore, deploymentStore dDS.DataStore, namespacesStore nsDS.DataStore, globalFlowStore flowStore.ClusterStore) Generator {
	return &generator{
		networkPolicies: networkPolicies,
		deploymentStore: deploymentStore,
		namespacesStore: namespacesStore,
		globalFlowStore: globalFlowStore,
	}
}
