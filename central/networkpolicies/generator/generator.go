package generator

import (
	"context"

	dDS "github.com/stackrox/stackrox/central/deployment/datastore"
	nsDS "github.com/stackrox/stackrox/central/namespace/datastore"
	networkBaselineDataStore "github.com/stackrox/stackrox/central/networkbaseline/datastore"
	"github.com/stackrox/stackrox/central/networkgraph/entity/networktree"
	nfDS "github.com/stackrox/stackrox/central/networkgraph/flow/datastore"
	npDS "github.com/stackrox/stackrox/central/networkpolicies/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
)

// Generator encapsulates the logic of the network policy generator.
type Generator interface {
	Generate(ctx context.Context, req *v1.GenerateNetworkPoliciesRequest) (generated []*storage.NetworkPolicy, toDelete []*storage.NetworkPolicyReference, err error)
	GenerateFromBaselineForDeployment(ctx context.Context, req *v1.GetBaselineGeneratedPolicyForDeploymentRequest) (generated []*storage.NetworkPolicy, toDelete []*storage.NetworkPolicyReference, err error)
}

// New creates and returns a new network policy generator.
func New(networkPolicies npDS.DataStore,
	deploymentStore dDS.DataStore,
	namespacesStore nsDS.DataStore,
	globalFlowDataStore nfDS.ClusterDataStore,
	networkTreeMgr networktree.Manager,
	networkBaselines networkBaselineDataStore.ReadOnlyDataStore,
) Generator {
	return &generator{
		networkPolicies:     networkPolicies,
		deploymentStore:     deploymentStore,
		namespacesStore:     namespacesStore,
		globalFlowDataStore: globalFlowDataStore,
		networkTreeMgr:      networkTreeMgr,
		networkBaselines:    networkBaselines,
	}
}
