package graph

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

// Evaluator implements the interface for the network graph generator
//go:generate mockgen-wrapper
type Evaluator interface {
	GetGraph(clusterID string, deployments []*storage.Deployment, networkPolicies []*storage.NetworkPolicy) *v1.NetworkGraph
	GetAppliedPolicies(deployments []*storage.Deployment, networkPolicies []*storage.NetworkPolicy) []*storage.NetworkPolicy
	IncrementEpoch(clusterID string)
	Epoch(clusterID string) uint32
}

type namespaceProvider interface {
	GetNamespaces() ([]*storage.NamespaceMetadata, error)
}

// evaluatorImpl handles all of the graph calculations
type evaluatorImpl struct {
	clusterEpochMap map[string]uint32
	epochMutex      sync.RWMutex

	namespaceStore namespaceProvider
}

// policyConnector represents a single policy and connects the policy to sets of nodes to which the policy is applied and which the policy matches.
type policyConnector struct {
	appliedIngress map[*v1.NetworkNode]struct{}
	matchedEgress  map[*v1.NetworkNode]struct{}
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
func (g *evaluatorImpl) GetGraph(clusterID string, deployments []*storage.Deployment, networkPolicies []*storage.NetworkPolicy) *v1.NetworkGraph {
	nodes := g.evaluate(deployments, networkPolicies)
	return &v1.NetworkGraph{
		Epoch: g.Epoch(clusterID),
		Nodes: nodes,
	}
}

// GetAppliedPolicies creates a filtered list of policies from the input network policies, composed of only the policies
// that apply to one or more of the input deployments.
func (g *evaluatorImpl) GetAppliedPolicies(deployments []*storage.Deployment, networkPolicies []*storage.NetworkPolicy) []*storage.NetworkPolicy {
	namespacesByID := g.getNamespacesByID()

	// For every deployment, determine the policies that apply to it.
	allApplied := set.NewStringSet()
	for _, deployment := range deployments {
		data := MatchDeploymentToPolicies(namespacesByID[deployment.GetNamespaceId()], deployment, networkPolicies)
		allApplied = allApplied.Union(data.appliedEgress.Union(data.appliedIngress))
	}
	if allApplied.IsEmpty() {
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
	namespacesByID := g.getNamespacesByID()

	// Create the nodes, with some extra data attached.
	nodes = make([]*v1.NetworkNode, 0, len(deployments))
	nodeToNodeData := make(map[*v1.NetworkNode]*DeploymentPolicyData, len(deployments))
	for _, deployment := range deployments {
		data := MatchDeploymentToPolicies(namespacesByID[deployment.GetNamespaceId()], deployment, networkPolicies)
		node := createNode(deployment, data)

		nodes = append(nodes, node)
		nodeToNodeData[node] = data
	}

	// Use the indices to fill in the outgoing edges.
	setOutgoingEdges(nodes, nodeToNodeData)
	return nodes
}

func (g *evaluatorImpl) getNamespacesByID() map[string]*storage.NamespaceMetadata {
	namespaces, err := g.namespaceStore.GetNamespaces()
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

func createNode(deployment *storage.Deployment, dpd *DeploymentPolicyData) *v1.NetworkNode {
	// If there are no egress policies, then it defaults to true
	if dpd.appliedEgress.IsEmpty() {
		dpd.internetAccess = true
	}

	// Combine applied policies for the node.
	nodePoliciesSet := dpd.appliedIngress.Union(dpd.appliedEgress).AsSortedSlice(func(i, j string) bool {
		return i < j
	})

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

func makePolicyConnectors(nodes []*v1.NetworkNode, nodeToNodeData map[*v1.NetworkNode]*DeploymentPolicyData) map[string]*policyConnector {
	policyConnectors := make(map[string]*policyConnector)
	for _, node := range nodes {
		nodeData := nodeToNodeData[node]
		for _, policyID := range nodeData.appliedIngress.AsSlice() {
			connector := getOrCreatePolicyConnector(policyID, policyConnectors)
			connector.appliedIngress[node] = struct{}{}
		}
		for _, policyID := range nodeData.matchedEgress.AsSlice() {
			connector := getOrCreatePolicyConnector(policyID, policyConnectors)
			connector.matchedEgress[node] = struct{}{}
		}
	}
	return policyConnectors
}

func getOrCreatePolicyConnector(policyID string, policyConnectors map[string]*policyConnector) *policyConnector {
	conn := policyConnectors[policyID]
	if conn == nil {
		conn = newPolicyConnector()
		policyConnectors[policyID] = conn
	}
	return conn
}

func newPolicyConnector() *policyConnector {
	return &policyConnector{
		appliedIngress: make(map[*v1.NetworkNode]struct{}),
		matchedEgress:  make(map[*v1.NetworkNode]struct{}),
	}
}

func setOutgoingEdges(nodes []*v1.NetworkNode, nodeToNodeData map[*v1.NetworkNode]*DeploymentPolicyData) {
	// Build maps of policies to the nodes they affect
	policyConnectors := makePolicyConnectors(nodes, nodeToNodeData)
	// All nodes mapped to their indices so we can find the index of destinations when we iterate out of order
	indexMap := make(map[*v1.NetworkNode]int, len(nodes))
	// Pre-compute the set of all nodes without ingress policies because we might be reusing it
	receiveFromAll := make(map[*v1.NetworkNode]struct{}, len(nodes))
	for i, node := range nodes {
		indexMap[node] = i
		nodeData := nodeToNodeData[node]
		if nodeData.appliedIngress.IsEmpty() {
			receiveFromAll[node] = struct{}{}
		}
	}
	for srcNode, srcData := range nodeToNodeData {
		srcNode.NonIsolatedIngress = srcData.appliedIngress.IsEmpty()
		srcNode.NonIsolatedEgress = srcData.appliedEgress.IsEmpty()

		// The set of nodes with applied ingress policies which allow them to receive from srcNode
		allowedIngress := getAllowedIngress(srcData, policyConnectors)

		if srcData.appliedEgress.IsEmpty() {
			// No egress policies, therefore the set of allowed edges is the set of nodes allowed to receive from srcNode
			for dstNode := range allowedIngress {
				if dstNode == srcNode {
					continue
				}
				srcNode.OutEdges[int32(indexMap[dstNode])] = &v1.NetworkEdgePropertiesBundle{}
			}
		}

		// The set of nodes srcNode's applied egress policies allow srcNode to transmit to.
		allowedEgress := getAllowedEgress(srcData, policyConnectors)
		// Find the intersection of allowed egress and allowed ingress including nodes with no ingress policies
		for dstNode := range allowedEgress {
			if dstNode == srcNode {
				continue
			}
			_, inAllowed := allowedIngress[dstNode]
			_, inReceiveFromAll := receiveFromAll[dstNode]
			if inAllowed || inReceiveFromAll {
				srcNode.OutEdges[int32(indexMap[dstNode])] = &v1.NetworkEdgePropertiesBundle{}
			}
		}
	}
}

func getAllowedEgress(data *DeploymentPolicyData, policyConnectors map[string]*policyConnector) map[*v1.NetworkNode]struct{} {
	return getAllowedNodes(data.appliedEgress.AsSlice(), policyConnectors, func(policy *policyConnector) map[*v1.NetworkNode]struct{} { return policy.matchedEgress })
}

func getAllowedIngress(data *DeploymentPolicyData, policyConnectors map[string]*policyConnector) map[*v1.NetworkNode]struct{} {
	return getAllowedNodes(data.matchedIngress.AsSlice(), policyConnectors, func(policy *policyConnector) map[*v1.NetworkNode]struct{} { return policy.appliedIngress })
}

func getAllowedNodes(policyIDs []string, policyConnectors map[string]*policyConnector, setFromPolicy func(*policyConnector) map[*v1.NetworkNode]struct{}) map[*v1.NetworkNode]struct{} {
	dstNodes := make(map[*v1.NetworkNode]struct{})
	for _, policyID := range policyIDs {
		if policy, ok := policyConnectors[policyID]; ok {
			for dstNode := range setFromPolicy(policy) {
				dstNodes[dstNode] = struct{}{}
			}
		}
	}
	return dstNodes
}
