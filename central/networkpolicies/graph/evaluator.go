package graph

import (
	"sort"
	"sync/atomic"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()
)

// Evaluator implements the interface for the network graph generator
//go:generate mockgen-wrapper Evaluator
type Evaluator interface {
	GetGraph(deployments []*storage.Deployment, networkPolicies []*storage.NetworkPolicy) *v1.NetworkGraph
	GetAppliedPolicies(deployments []*storage.Deployment, networkPolicies []*storage.NetworkPolicy) []*storage.NetworkPolicy
	IncrementEpoch()
	Epoch() uint32
}

type namespaceProvider interface {
	GetNamespaces() ([]*storage.NamespaceMetadata, error)
}

// evaluatorImpl handles all of the graph calculations
type evaluatorImpl struct {
	epoch uint32

	deploymentMatcher *deploymentMatcher
}

// newGraphEvaluator takes in namespaces
func newGraphEvaluator(namespaceStore namespaceProvider) *evaluatorImpl {
	return &evaluatorImpl{
		deploymentMatcher: newDeploymentPolicyMatcher(namespaceStore),
	}
}

// IncrementEpoch increments epoch, effectively indicating that a graph that is generated may change.
func (g *evaluatorImpl) IncrementEpoch() {
	atomic.AddUint32(&g.epoch, 1)
}

// Epoch returns the current value for epoch, which tracks modifications to deployments.
func (g *evaluatorImpl) Epoch() uint32 {
	return atomic.LoadUint32(&g.epoch)
}

// GetGraph generates a network graph for the input deployments based on the input policies.
func (g *evaluatorImpl) GetGraph(deployments []*storage.Deployment, networkPolicies []*storage.NetworkPolicy) *v1.NetworkGraph {
	nodes := g.evaluate(deployments, networkPolicies)
	return &v1.NetworkGraph{
		Epoch: g.Epoch(),
		Nodes: nodes,
	}
}

// GetAppliedPolicies creates a filtered list of policies from the input network policies, composed of only the policies
// that apply to one or more of the input deployments.
func (g *evaluatorImpl) GetAppliedPolicies(deployments []*storage.Deployment, networkPolicies []*storage.NetworkPolicy) []*storage.NetworkPolicy {
	// For every deployment, determine the policies that apply to it.
	allApplied := set.NewStringSet()
	for _, deployment := range deployments {
		data := g.deploymentMatcher.MatchDeploymentToPolicies(deployment, networkPolicies)
		allApplied = allApplied.Union(data.appliedEgress.Union(data.appliedIngress))
	}
	if allApplied.Cardinality() == 0 {
		return nil
	}

	// Create a new list of policies composed of only the ones applied to a deployment.
	appliedPolicies := make([]*storage.NetworkPolicy, 0, allApplied.Cardinality())
	for _, policy := range networkPolicies {
		if allApplied.Contains(policy.GetId()) {
			appliedPolicies = append(appliedPolicies, policy)
		}
	}
	return appliedPolicies
}

func (g *evaluatorImpl) evaluate(deployments []*storage.Deployment, networkPolicies []*storage.NetworkPolicy) (nodes []*v1.NetworkNode) {
	// Create the nodes, with some extra data attached.
	nodes = make([]*v1.NetworkNode, 0, len(deployments))
	nodeToNodeData := make(map[*v1.NetworkNode]*DeploymentPolicyData, len(deployments))
	for _, deployment := range deployments {
		data := g.deploymentMatcher.MatchDeploymentToPolicies(deployment, networkPolicies)
		node := createNode(deployment, data)

		nodes = append(nodes, node)
		nodeToNodeData[node] = data
	}

	// Use the indices to fill in the outgoing edges.
	setOutgoingEdges(nodes, nodeToNodeData)
	return nodes
}

func createNode(deployment *storage.Deployment, dpd *DeploymentPolicyData) *v1.NetworkNode {
	// If there are no egress policies, then it defaults to true
	if dpd.appliedEgress.Cardinality() == 0 {
		dpd.internetAccess = true
	}

	// Combine applied policies for the node.
	nodePoliciesSet := dpd.appliedIngress.Union(dpd.appliedEgress).AsSortedSlice(func(i, j string) bool {
		return i < j
	})
	sort.Strings(nodePoliciesSet)

	// Create and return the node.
	return &v1.NetworkNode{
		Entity: &storage.NetworkEntityInfo{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			Id:   deployment.GetId(),
			Desc: &storage.NetworkEntityInfo_Deployment_{
				Deployment: &storage.NetworkEntityInfo_Deployment{
					Name:      deployment.GetName(),
					Namespace: deployment.GetNamespace(),
					Cluster:   deployment.GetClusterName(),
				},
			},
		},
		InternetAccess: dpd.internetAccess,
		PolicyIds:      nodePoliciesSet,
		OutEdges:       make(map[int32]*v1.NetworkEdgePropertiesBundle),
	}
}

func setOutgoingEdges(nodes []*v1.NetworkNode, nodeToNodeData map[*v1.NetworkNode]*DeploymentPolicyData) {
	for srcIndex, srcNode := range nodes {
		srcData := nodeToNodeData[srcNode]
		srcNode.NonIsolatedIngress = srcData.appliedIngress.Cardinality() == 0
		srcNode.NonIsolatedEgress = srcData.appliedEgress.Cardinality() == 0

		for dstIndex, dstNode := range nodes {
			if srcIndex == dstIndex {
				continue
			}
			dstData := nodeToNodeData[dstNode]

			// Only add edges that are either due to an ingress policy on the destination side, or an egress policy on
			// the source side.
			if srcData.appliedEgress.Cardinality() == 0 && dstData.appliedIngress.Cardinality() == 0 {
				continue
			}

			// This set is the set of Egress policies that are applicable to the src
			selectedEgressPoliciesSet := srcData.appliedEgress
			// This set is the set if Egress policies that have rules that are applicable to the dst
			matchedEgressPoliciesSet := dstData.matchedEgress
			// If there are no values in the src set of egress then it has no Egress rules and can talk to everything
			// Otherwise, if it is not empty then ensure that the intersection of the policies that apply to the source and the rules that apply to the dst have at least one in common
			if selectedEgressPoliciesSet.Cardinality() != 0 && selectedEgressPoliciesSet.Intersect(matchedEgressPoliciesSet).Cardinality() == 0 {
				continue
			}

			// This set is the set of Ingress policies that are applicable to the dst
			selectedIngressPoliciesSet := dstData.appliedIngress
			// This set is the set if Ingress policies that have rules that are applicable to the src
			matchedIngressPoliciesSet := srcData.matchedIngress
			// If there are no values in the src set of egress then it has no Egress rules and can talk to everything
			// Otherwise, if it is not empty then ensure that the intersection of the policies that apply to the source and the rules that apply to the dst have at least one in common
			if selectedIngressPoliciesSet.Cardinality() != 0 && selectedIngressPoliciesSet.Intersect(matchedIngressPoliciesSet).Cardinality() == 0 {
				continue
			}

			srcNode.OutEdges[int32(dstIndex)] = &v1.NetworkEdgePropertiesBundle{}
		}
	}
}
