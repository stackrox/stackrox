package graph

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	allNamespaceReadAccess = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Namespace),
		),
	)

	log = logging.LoggerForModule()
)

// Evaluator implements the interface for the network graph generator
//
//go:generate mockgen-wrapper
type Evaluator interface {
	// GetGraph returns the network policy graph. If `queryDeploymentIDs` is nil, it is assumed that all deployments are queried/relevant.
	GetGraph(clusterID string, queryDeploymentIDs set.StringSet, clusterDeployments []*storage.Deployment, networkTree tree.ReadOnlyNetworkTree, networkPolicies []*storage.NetworkPolicy, includePorts bool) *v1.NetworkGraph
	GetAppliedPolicies(deployments []*storage.Deployment, networkTree tree.ReadOnlyNetworkTree, networkPolicies []*storage.NetworkPolicy) []*storage.NetworkPolicy
	GetApplyingPoliciesPerDeployment(deployments []*storage.Deployment, networkTree tree.ReadOnlyNetworkTree, networkPolicies []*storage.NetworkPolicy) map[string][]*storage.NetworkPolicy
	IncrementEpoch(clusterID string)
	Epoch(clusterID string) uint32
}

type namespaceProvider interface {
	GetAllNamespaces(ctx context.Context) ([]*storage.NamespaceMetadata, error)
}

// evaluatorImpl handles all of the graph calculations
type evaluatorImpl struct {
	clusterEpochMap map[string]uint32
	epochMutex      sync.RWMutex

	namespaceStore namespaceProvider
}

// newGraphEvaluator takes in namespaces
func newGraphEvaluator(namespaceStore namespaceProvider) *evaluatorImpl {
	return &evaluatorImpl{
		namespaceStore:  namespaceStore,
		clusterEpochMap: make(map[string]uint32),
	}
}

// IncrementEpoch increments epoch, effectively indicating that a graph that is generated may change.
func (g *evaluatorImpl) IncrementEpoch(clusterID string) {
	g.epochMutex.Lock()
	defer g.epochMutex.Unlock()
	g.clusterEpochMap[clusterID]++
}

// Epoch returns the current value for epoch, which tracks modifications to deployments.
func (g *evaluatorImpl) Epoch(clusterID string) uint32 {
	g.epochMutex.RLock()
	defer g.epochMutex.RUnlock()
	if clusterID != "" {
		return g.clusterEpochMap[clusterID]
	}
	var totalEpoch uint32
	for _, v := range g.clusterEpochMap {
		totalEpoch += v
	}
	return totalEpoch
}

// GetGraph generates a network graph for the input deployments based on the input policies.
func (g *evaluatorImpl) GetGraph(clusterID string, queryDeploymentIDs set.StringSet, clusterDeployments []*storage.Deployment, networkTree tree.ReadOnlyNetworkTree, networkPolicies []*storage.NetworkPolicy, includePorts bool) *v1.NetworkGraph {
	namespacesByID := g.getNamespacesByID()

	b := newGraphBuilder(queryDeploymentIDs, clusterDeployments, networkTree, namespacesByID)
	b.AddEdgesForNetworkPolicies(networkPolicies)
	b.PostProcess()
	nodes := b.ToProto(includePorts)
	return &v1.NetworkGraph{
		Epoch: g.Epoch(clusterID),
		Nodes: nodes,
	}
}

// GetApplyingPoliciesPerDeployment creates a map of deployment IDs to the applying network policies
func (g *evaluatorImpl) GetApplyingPoliciesPerDeployment(deployments []*storage.Deployment, networkTree tree.ReadOnlyNetworkTree, networkPolicies []*storage.NetworkPolicy) map[string][]*storage.NetworkPolicy {
	return newGraphBuilder(nil, deployments, networkTree, g.getNamespacesByID()).GetApplyingPoliciesPerDeployment(networkPolicies)
}

// GetAppliedPolicies creates a filtered list of policies from the input network policies, composed of only the policies
// that apply to one or more of the input deployments.
func (g *evaluatorImpl) GetAppliedPolicies(deployments []*storage.Deployment, networkTree tree.ReadOnlyNetworkTree, networkPolicies []*storage.NetworkPolicy) []*storage.NetworkPolicy {
	return newGraphBuilder(nil, deployments, networkTree, g.getNamespacesByID()).GetApplyingPolicies(networkPolicies)
}

func (g *evaluatorImpl) getNamespacesByID() map[string]*storage.NamespaceMetadata {
	namespaces, err := g.namespaceStore.GetAllNamespaces(allNamespaceReadAccess)
	if err != nil {
		log.Errorf("unable to read namespaces: %v", err)
		return nil
	}

	namespacesByID := make(map[string]*storage.NamespaceMetadata)
	for _, namespace := range namespaces {
		namespacesByID[namespace.GetId()] = namespace
	}
	return namespacesByID
}
