package networkgraph

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"

	clusterDatastore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	deploymentsDatastore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	networkPolicyStore "bitbucket.org/stack-rox/apollo/central/networkpolicies/store"
)

var logger = logging.LoggerForModule()

// GraphEvaluator implements the interface for the network graph generator
type GraphEvaluator interface {
	GetGraph() (*v1.GetNetworkGraphResponse, error)
}

// graphEvaluatorImpl handles all of the graph calculations
type graphEvaluatorImpl struct {
	clustersStore      clusterDatastore.DataStore
	deploymentsStore   deploymentsDatastore.DataStore
	namespaceStore     ns
	networkPolicyStore networkPolicyStore.Store
}

type ns interface {
	GetNamespaces() ([]*v1.Namespace, error)
}

// newGraphEvaluator takes in namespaces
func newGraphEvaluator(clustersStore clusterDatastore.DataStore, deploymentsStore deploymentsDatastore.DataStore,
	namespaceStore ns, networkPolicyStore networkPolicyStore.Store) *graphEvaluatorImpl {
	return &graphEvaluatorImpl{
		clustersStore:      clustersStore,
		deploymentsStore:   deploymentsStore,
		namespaceStore:     namespaceStore,
		networkPolicyStore: networkPolicyStore,
	}
}

func (g *graphEvaluatorImpl) GetGraph() (*v1.GetNetworkGraphResponse, error) {
	nodes, edges, err := g.evaluate()
	if err != nil {
		return nil, err
	}
	return &v1.GetNetworkGraphResponse{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func (g *graphEvaluatorImpl) evaluate() (nodes []*v1.NetworkNode, edges []*v1.NetworkEdge, err error) {
	var graphGroupNum int32
	graphGroup := make(map[string]int32)

	clusters, err := g.clustersStore.GetClusters()
	if err != nil {
		return
	}

	networkPolicies, err := g.networkPolicyStore.GetNetworkPolicies()
	if err != nil {
		return
	}

	for _, c := range clusters {
		var deployments []*v1.Deployment
		deployments, err = g.deploymentsStore.SearchRawDeployments(&v1.ParsedSearchRequest{
			Scopes: []*v1.Scope{
				{
					Cluster: c.GetName(),
				},
			},
		})
		if err != nil {
			return
		}

		for _, d := range deployments {
			val, ok := graphGroup[c.GetName()+"-"+d.GetNamespace()]
			if !ok {
				graphGroupNum++
				graphGroup[c.GetName()+"-"+d.GetNamespace()] = graphGroupNum
				val = graphGroupNum
			}
			nodes = append(nodes, &v1.NetworkNode{Id: d.GetId(), Group: val})
		}
		edges = append(g.evaluateCluster(deployments, networkPolicies))
	}
	return
}

func (g *graphEvaluatorImpl) evaluateCluster(deployments []*v1.Deployment, networkPolicies []*v1.NetworkPolicy) []*v1.NetworkEdge {
	var edges []*v1.NetworkEdge

	for i1, d1 := range deployments {
		for i2, d2 := range deployments {
			if i1 == i2 {
				continue
			}
			if policyNames, hasEdge := g.evaluateDeploymentPair(d1, d2, networkPolicies); hasEdge {
				edges = append(edges, &v1.NetworkEdge{Source: d1.GetId(), Target: d2.GetId(), PolicyNames: policyNames, Value: 1})
			}
		}
	}
	return edges
}

func (g *graphEvaluatorImpl) getNamespace(deployment *v1.Deployment) *v1.Namespace {
	namespaces, err := g.namespaceStore.GetNamespaces()
	if err != nil {
		return &v1.Namespace{
			Name: deployment.GetNamespace(),
		}
	}
	for _, n := range namespaces {
		if n.GetName() == deployment.GetNamespace() && n.GetClusterId() == deployment.GetClusterId() {
			return n
		}
	}
	return &v1.Namespace{
		Name: deployment.GetNamespace(),
	}
}

func (g *graphEvaluatorImpl) matchPolicyPeer(deployment *v1.Deployment, policyNamespace string, peer *v1.NetworkPolicyPeer) bool {
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

func (g *graphEvaluatorImpl) evaluateDeploymentPairWithEgressNetworkPolicy(src, dst *v1.Deployment, np *v1.NetworkPolicy) (validEgress bool, match bool) {
	spec := np.Spec

	if spec == nil {
		return
	}

	// Check if the src matches the pod selector and deployment then the egress rules actually apply to that deployment
	if !doesPodLabelsMatchLabel(src, spec.GetPodSelector()) || src.GetNamespace() != np.GetNamespace() {
		return false, false
	}
	// If no egress rules are defined, then it doesn't apply
	if !hasEgress(spec.PolicyTypes) {
		return false, false
	}
	validEgress = true
	for _, egressRule := range spec.Egress {
		if len(egressRule.To) == 0 {
			match = true
			return
		}
		for _, f := range egressRule.To {
			if g.matchPolicyPeer(dst, np.GetNamespace(), f) {
				match = true
				return
			}
		}
	}
	return
}

func (g *graphEvaluatorImpl) evaluateDeploymentPairWithIngressNetworkPolicy(src, dst *v1.Deployment, np *v1.NetworkPolicy) (valid bool, match bool) {
	spec := np.Spec
	if spec == nil {
		return
	}

	// Check if the src matches the pod selector and deployment then the egress rules actually apply to that deployment
	if !doesPodLabelsMatchLabel(dst, spec.PodSelector) || dst.GetNamespace() != np.GetNamespace() {
		return false, false
	}
	// If no egress rules are defined, then it doesn't apply
	if !hasIngress(spec.PolicyTypes) {
		return false, false
	}
	valid = true
	for _, ingressRule := range spec.Ingress {
		// If there is a rule with no values then it matches all
		// This in contrast to if there are no spec.Ingress defined which blocks all
		// Yeah, not very clear :(
		if len(ingressRule.From) == 0 {
			match = true
			return
		}
		// TODO(cgorman) write test case for this function for this particular case :(
		for _, f := range ingressRule.From {
			if g.matchPolicyPeer(src, np.GetNamespace(), f) {
				match = true
				return
			}
		}
	}
	return
}

func (g *graphEvaluatorImpl) evaluateDeploymentPairIngress(src, dst *v1.Deployment, policies []*v1.NetworkPolicy) ([]string, bool) {
	var policyNames []string
	var ingressDefined bool
	for _, np := range policies {
		if validIngress, match := g.evaluateDeploymentPairWithIngressNetworkPolicy(src, dst, np); validIngress {
			ingressDefined = true
			if match {
				policyNames = append(policyNames, np.GetName())
			}
		}
	}

	if len(policyNames) != 0 {
		return policyNames, true
	}

	// If ingress is defined, but there was no match then ingress is not allowed
	// If no ingress was defined, then ingress is allowed by default
	return policyNames, !ingressDefined
}

func (g *graphEvaluatorImpl) evaluateDeploymentPairEgress(src, dst *v1.Deployment, policies []*v1.NetworkPolicy) ([]string, bool) {
	var policyNames []string

	var egressDefined bool
	for _, np := range policies {
		if validEgress, match := g.evaluateDeploymentPairWithEgressNetworkPolicy(src, dst, np); validEgress {
			egressDefined = true
			if match {
				policyNames = append(policyNames, string(np.GetName()))
			}
		}
	}

	if len(policyNames) != 0 {
		return policyNames, true
	}

	// If egress is defined, but there was no match then egress is not allowed
	// If no egress was defined, then egress is allowed by default
	return policyNames, !egressDefined
}

func (g *graphEvaluatorImpl) evaluateDeploymentPair(d1, d2 *v1.Deployment, networkPolicies []*v1.NetworkPolicy) ([]string, bool) {
	ingressPolicyIDs, ingressMatch := g.evaluateDeploymentPairIngress(d1, d2, networkPolicies)
	if !ingressMatch {
		return nil, false
	}

	egressPolicyIDs, egressMatch := g.evaluateDeploymentPairEgress(d1, d2, networkPolicies)
	if !egressMatch {
		return nil, false
	}
	return append(ingressPolicyIDs, egressPolicyIDs...), true
}

func doesNamespaceMatchLabel(namespace *v1.Namespace, selector *v1.LabelSelector) bool {
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

func doesPodLabelsMatchLabel(deployment *v1.Deployment, podSelector *v1.LabelSelector) bool {
	// No values equals match all
	if len(podSelector.GetMatchLabels()) == 0 {
		return true
	}
	deploymentLabelMap := make(map[string]string)
	for _, keyValue := range deployment.GetLabels() {
		deploymentLabelMap[keyValue.GetKey()] = keyValue.GetValue()
	}
	for k, v := range podSelector.GetMatchLabels() {
		if deploymentLabelMap[k] != v {
			return false
		}
	}
	return true
}

func hasPolicyType(types []v1.NetworkPolicyType, t v1.NetworkPolicyType) bool {
	for _, pType := range types {
		if pType == t {
			return true
		}
	}
	return false
}

func hasEgress(types []v1.NetworkPolicyType) bool {
	return hasPolicyType(types, v1.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE)
}

func hasIngress(types []v1.NetworkPolicyType) bool {
	if len(types) == 0 {
		return true
	}
	return hasPolicyType(types, v1.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE)
}
