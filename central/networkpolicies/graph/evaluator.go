package graph

import (
	"sort"
	"sync/atomic"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

var logger = logging.LoggerForModule()

// Evaluator implements the interface for the network graph generator
//go:generate mockgen-wrapper Evaluator
type Evaluator interface {
	GetGraph(deployments []*storage.Deployment, networkPolicies []*storage.NetworkPolicy) *v1.NetworkGraph
	IncrementEpoch()
	Epoch() uint32
}

type namespaceProvider interface {
	GetNamespaces() ([]*storage.Namespace, error)
}

// evaluatorImpl handles all of the graph calculations
type evaluatorImpl struct {
	epoch uint32

	namespaceStore namespaceProvider
}

// newGraphEvaluator takes in namespaces
func newGraphEvaluator(namespaceStore namespaceProvider) *evaluatorImpl {
	return &evaluatorImpl{
		namespaceStore: namespaceStore,
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

type nodeData struct {
	selectedDeploymentsToIngressPolicies set.StringSet
	selectedDeploymentsToEgressPolicies  set.StringSet
	matchedDeploymentsToIngressPolicies  set.StringSet
	matchedDeploymentsToEgressPolicies   set.StringSet

	node  *v1.NetworkNode
	index int
}

func (g *evaluatorImpl) evaluate(deployments []*storage.Deployment, networkPolicies []*storage.NetworkPolicy) []*v1.NetworkNode {
	nodeDataMap := make(map[*storage.Deployment]*nodeData, len(deployments))
	nodes := make([]*v1.NetworkNode, 0, len(deployments))

	for i, d := range deployments {
		data := &nodeData{
			selectedDeploymentsToIngressPolicies: set.NewStringSet(),
			selectedDeploymentsToEgressPolicies:  set.NewStringSet(),
			matchedDeploymentsToIngressPolicies:  set.NewStringSet(),
			matchedDeploymentsToEgressPolicies:   set.NewStringSet(),

			index: i,
		}

		var internetAccess bool
		for _, n := range networkPolicies {
			if n.GetSpec() == nil {
				continue
			}
			if ingressNetworkPolicySelectorAppliesToDeployment(d, n) {
				data.selectedDeploymentsToIngressPolicies.Add(n.GetId())
			}
			if g.doesIngressNetworkPolicyRuleMatchDeployment(d, n) {
				data.matchedDeploymentsToIngressPolicies.Add(n.GetId())
			}
			if applies, internetConnection := egressNetworkPolicySelectorAppliesToDeployment(d, n); applies {
				data.selectedDeploymentsToEgressPolicies.Add(n.GetId())
				if internetConnection {
					internetAccess = true
				}
			}
			if g.doesEgressNetworkPolicyRuleMatchDeployment(d, n) {
				data.matchedDeploymentsToEgressPolicies.Add(n.GetId())
			}
		}
		// If there are no egress policies, then it defaults to true
		if data.selectedDeploymentsToEgressPolicies.Cardinality() == 0 {
			internetAccess = true
		}

		nodePoliciesSet := data.selectedDeploymentsToIngressPolicies.Union(data.selectedDeploymentsToEgressPolicies).AsSlice()
		sort.Strings(nodePoliciesSet)

		node := &v1.NetworkNode{
			Entity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   d.GetId(),
				Desc: &storage.NetworkEntityInfo_Deployment_{
					Deployment: &storage.NetworkEntityInfo_Deployment{
						Name:      d.GetName(),
						Namespace: d.GetNamespace(),
						Cluster:   d.GetClusterName(),
					},
				},
			},
			InternetAccess: internetAccess,
			PolicyIds:      nodePoliciesSet,
			OutEdges:       make(map[int32]*v1.NetworkEdgePropertiesBundle),
		}

		data.node = node
		nodeDataMap[d] = data

		nodes = append(nodes, node)
	}

	for _, src := range deployments {
		srcData := nodeDataMap[src]
		srcNode := srcData.node
		for _, dst := range deployments {
			if src == dst {
				continue
			}

			dstData := nodeDataMap[dst]

			// This set is the set of Egress policies that are applicable to the src
			selectedEgressPoliciesSet := srcData.selectedDeploymentsToEgressPolicies
			// This set is the set if Egress policies that have rules that are applicable to the dst
			matchedEgressPoliciesSet := dstData.matchedDeploymentsToEgressPolicies
			// If there are no values in the src set of egress then it has no Egress rules and can talk to everything
			// Otherwise, if it is not empty then ensure that the intersection of the policies that apply to the source and the rules that apply to the dst have at least one in common
			if selectedEgressPoliciesSet.Cardinality() != 0 && selectedEgressPoliciesSet.Intersect(matchedEgressPoliciesSet).Cardinality() == 0 {
				continue
			}

			// This set is the set of Ingress policies that are applicable to the dst
			selectedIngressPoliciesSet := dstData.selectedDeploymentsToIngressPolicies
			// This set is the set if Ingress policies that have rules that are applicable to the src
			matchedIngressPoliciesSet := srcData.matchedDeploymentsToIngressPolicies
			// If there are no values in the src set of egress then it has no Egress rules and can talk to everything
			// Otherwise, if it is not empty then ensure that the intersection of the policies that apply to the source and the rules that apply to the dst have at least one in common
			if selectedIngressPoliciesSet.Cardinality() != 0 && selectedIngressPoliciesSet.Intersect(matchedIngressPoliciesSet).Cardinality() == 0 {
				continue
			}

			srcNode.OutEdges[int32(dstData.index)] = &v1.NetworkEdgePropertiesBundle{}
		}
	}
	return nodes
}

func egressNetworkPolicySelectorAppliesToDeployment(d *storage.Deployment, np *storage.NetworkPolicy) (applies bool, internetAccess bool) {
	spec := np.GetSpec()
	// Check if the src matches the pod selector and deployment then the egress rules actually apply to that deployment
	if !doesPodLabelsMatchLabel(d, spec.GetPodSelector()) || d.GetNamespace() != np.GetNamespace() {
		return
	}
	// If no egress rules are defined, then it doesn't apply
	if applies = hasEgress(spec.GetPolicyTypes()); !applies {
		return
	}

	// If there is a rule with an IPBlock that is not nil, then we can assume that they have some sort of internet access
	// This isn't exactly full proof, but probably a pretty decent indicator
	for _, rule := range spec.GetEgress() {
		for _, to := range rule.GetTo() {
			if to.IpBlock != nil {
				internetAccess = true
				return
			}
		}
	}
	return
}

func ingressNetworkPolicySelectorAppliesToDeployment(d *storage.Deployment, np *storage.NetworkPolicy) bool {
	spec := np.GetSpec()
	// Check if the src matches the pod selector and deployment then the egress rules actually apply to that deployment
	if !doesPodLabelsMatchLabel(d, spec.GetPodSelector()) || d.GetNamespace() != np.GetNamespace() {
		return false
	}
	// If no egress rules are defined, then it doesn't apply
	return hasIngress(spec.GetPolicyTypes())
}

func (g *evaluatorImpl) doesEgressNetworkPolicyRuleMatchDeployment(src *storage.Deployment, np *storage.NetworkPolicy) bool {
	for _, egressRule := range np.GetSpec().GetEgress() {
		if g.matchPolicyPeers(src, np.GetNamespace(), egressRule.GetTo()) {
			return true
		}
	}
	return false
}

func (g *evaluatorImpl) doesIngressNetworkPolicyRuleMatchDeployment(src *storage.Deployment, np *storage.NetworkPolicy) bool {
	for _, ingressRule := range np.GetSpec().GetIngress() {
		if g.matchPolicyPeers(src, np.GetNamespace(), ingressRule.GetFrom()) {
			return true
		}
	}
	return false
}

func (g *evaluatorImpl) matchPolicyPeers(d *storage.Deployment, namespace string, peers []*storage.NetworkPolicyPeer) bool {
	if len(peers) == 0 {
		return true
	}
	for _, p := range peers {
		if g.matchPolicyPeer(d, namespace, p) {
			return true
		}
	}
	return false
}

func (g *evaluatorImpl) matchPolicyPeer(deployment *storage.Deployment, policyNamespace string, peer *storage.NetworkPolicyPeer) bool {
	if peer.IpBlock != nil {
		logger.Infof("IP Block network policy is currently not handled")
		return false
	}

	// If namespace selector is specified, then make sure the namespace matches
	// Other you fall back to the fact that the deployment must be in the policy's namespace
	if peer.GetNamespaceSelector() != nil {
		namespace := g.getNamespace(deployment)
		if !doesNamespaceMatchLabel(namespace, peer.GetNamespaceSelector()) {
			return false
		}
	} else if deployment.GetNamespace() != policyNamespace {
		return false
	}

	if peer.GetPodSelector() != nil {
		return doesPodLabelsMatchLabel(deployment, peer.GetPodSelector())
	}
	return true
}

func (g *evaluatorImpl) getNamespace(deployment *storage.Deployment) *storage.Namespace {
	namespaces, err := g.namespaceStore.GetNamespaces()
	if err != nil {
		return &storage.Namespace{
			Name: deployment.GetNamespace(),
		}
	}
	for _, n := range namespaces {
		if n.GetName() == deployment.GetNamespace() && n.GetClusterId() == deployment.GetClusterId() {
			return n
		}
	}
	return &storage.Namespace{
		Name: deployment.GetNamespace(),
	}
}

func doesNamespaceMatchLabel(namespace *storage.Namespace, selector *storage.LabelSelector) bool {
	if len(selector.MatchLabels) == 0 {
		return true
	}
	for k, v := range namespace.GetLabels() {
		if selector.MatchLabels[k] == v {
			return true
		}
	}
	return false
}

func doesPodLabelsMatchLabel(deployment *storage.Deployment, podSelector *storage.LabelSelector) bool {
	// No values equals match all
	if len(podSelector.GetMatchLabels()) == 0 {
		return true
	}
	for k, v := range podSelector.GetMatchLabels() {
		if deployment.GetLabels()[k] != v {
			return false
		}
	}
	return true
}

func hasEgress(types []storage.NetworkPolicyType) bool {
	return hasPolicyType(types, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE)
}

func hasIngress(types []storage.NetworkPolicyType) bool {
	if len(types) == 0 {
		return true
	}
	return hasPolicyType(types, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE)
}

func hasPolicyType(types []storage.NetworkPolicyType, t storage.NetworkPolicyType) bool {
	for _, pType := range types {
		if pType == t {
			return true
		}
	}
	return false
}
