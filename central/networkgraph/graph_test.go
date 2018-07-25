package networkgraph

import (
	"bytes"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	"github.com/stretchr/testify/assert"
	k8sV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var networkPolicyFixtures = map[string]*v1.NetworkPolicy{}

func init() {
	for _, policyYAML := range networkPolicyFixtureYAMLs {
		var k8sNp k8sV1.NetworkPolicy
		if err := yaml.NewYAMLToJSONDecoder(bytes.NewReader([]byte(policyYAML))).Decode(&k8sNp); err != nil {
			panic(err)
		}
		np := protoconv.KubernetesNetworkPolicyWrap{NetworkPolicy: &k8sNp}.ConvertNetworkPolicy()
		np.Id = k8sNp.GetName()
		networkPolicyFixtures[np.GetName()] = np
	}
}

type namespaceGetter struct{}

func (n *namespaceGetter) GetNamespaces() ([]*v1.Namespace, error) {
	return []*v1.Namespace{
		{
			Name: "default",
			Labels: map[string]string{
				"name": "default",
			},
		},
		{
			Name: "stackrox",
			Labels: map[string]string{
				"name": "stackrox",
			},
		},
		{
			Name: "other",
		},
	}, nil
}

func newMockGraphEvaluator() *graphEvaluatorImpl {
	return newGraphEvaluator(nil, nil, &namespaceGetter{}, nil)
}

func TestDoesNamespaceMatchLabel(t *testing.T) {
	cases := []struct {
		name      string
		namespace *v1.Namespace
		selector  *v1.LabelSelector
		expected  bool
	}{
		{
			name:      "No values in selector - no namespace labels",
			namespace: &v1.Namespace{},
			selector:  &v1.LabelSelector{},
			expected:  true,
		},
		{
			name:      "No values in selector - some namespace labels",
			namespace: &v1.Namespace{},
			selector:  &v1.LabelSelector{},
			expected:  true,
		},
		{
			name: "matching values in selector",
			namespace: &v1.Namespace{
				Labels: map[string]string{
					"hello": "hi",
				},
			},
			selector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					"hello": "hi",
				},
			},
			expected: true,
		},
		{
			name: "non matching values in selector",
			namespace: &v1.Namespace{
				Labels: map[string]string{
					"hello": "hi1",
				},
			},
			selector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					"hello": "hi",
				},
			},
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, doesNamespaceMatchLabel(c.namespace, c.selector))
		})
	}
}

func TestDoesPodLabelsMatchLabel(t *testing.T) {
	cases := []struct {
		name       string
		deployment *v1.Deployment
		selector   *v1.LabelSelector
		expected   bool
	}{
		{
			name:       "No values in selector - no deployment labels",
			deployment: &v1.Deployment{},
			selector:   &v1.LabelSelector{},
			expected:   true,
		},
		{
			name:       "No values in selector - some deployment labels",
			deployment: &v1.Deployment{},
			selector:   &v1.LabelSelector{},
			expected:   true,
		},
		{
			name: "matching values in selector",
			deployment: &v1.Deployment{
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "hello",
						Value: "hi",
					},
				},
			},
			selector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					"hello": "hi",
				},
			},
			expected: true,
		},
		{
			name: "non matching values in selector",
			deployment: &v1.Deployment{
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "hello",
						Value: "hi1",
					},
				},
			},
			selector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					"hello": "hi",
				},
			},
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, doesPodLabelsMatchLabel(c.deployment, c.selector))
		})
	}
}

func TestHasEgress(t *testing.T) {
	cases := []struct {
		name        string
		policyTypes []v1.NetworkPolicyType
		expected    bool
	}{
		{
			name:        "no values",
			policyTypes: []v1.NetworkPolicyType{},
			expected:    false,
		},
		{
			name:        "ingress only",
			policyTypes: []v1.NetworkPolicyType{v1.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
			expected:    false,
		},
		{
			name:        "egress only",
			policyTypes: []v1.NetworkPolicyType{v1.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
			expected:    true,
		},
		{
			name:        "ingress + egress only",
			policyTypes: []v1.NetworkPolicyType{v1.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, v1.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
			expected:    true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, hasEgress(c.policyTypes))
		})
	}
}

func TestHasIngress(t *testing.T) {
	cases := []struct {
		name        string
		policyTypes []v1.NetworkPolicyType
		expected    bool
	}{
		{
			name:        "no values",
			policyTypes: []v1.NetworkPolicyType{},
			expected:    true,
		},
		{
			name:        "ingress only",
			policyTypes: []v1.NetworkPolicyType{v1.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
			expected:    true,
		},
		{
			name:        "egress only",
			policyTypes: []v1.NetworkPolicyType{v1.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
			expected:    false,
		},
		{
			name:        "ingress + egress only",
			policyTypes: []v1.NetworkPolicyType{v1.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, v1.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
			expected:    true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, hasIngress(c.policyTypes))
		})
	}
}

func TestMatchPolicyPeer(t *testing.T) {
	g := newMockGraphEvaluator()

	cases := []struct {
		name            string
		deployment      *v1.Deployment
		peer            *v1.NetworkPolicyPeer
		policyNamespace string
		expected        bool
	}{
		{
			name:       "ip block",
			deployment: &v1.Deployment{},
			peer:       &v1.NetworkPolicyPeer{IpBlock: &v1.IPBlock{}},
			expected:   false,
		},
		{
			name: "non match pod selector",
			deployment: &v1.Deployment{
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key",
						Value: "value1",
					},
				},
			},
			peer: &v1.NetworkPolicyPeer{
				PodSelector: &v1.LabelSelector{
					MatchLabels: map[string]string{
						"key": "value",
					},
				},
			},
			expected: false,
		},
		{
			name: "match pod selector",
			deployment: &v1.Deployment{
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key",
						Value: "value",
					},
				},
			},
			peer: &v1.NetworkPolicyPeer{
				PodSelector: &v1.LabelSelector{
					MatchLabels: map[string]string{
						"key": "value",
					},
				},
			},
			expected: true,
		},
		{
			name: "match namespace selector",
			deployment: &v1.Deployment{
				Namespace: "default",
			},
			peer: &v1.NetworkPolicyPeer{
				NamespaceSelector: &v1.LabelSelector{
					MatchLabels: map[string]string{
						"name": "default",
					},
				},
			},
			policyNamespace: "default",
			expected:        true,
		},
		{
			name: "non match namespace selector",
			deployment: &v1.Deployment{
				Namespace: "default",
			},
			peer: &v1.NetworkPolicyPeer{
				NamespaceSelector: &v1.LabelSelector{
					MatchLabels: map[string]string{
						"key": "value1",
					},
				},
			},
			policyNamespace: "default",
			expected:        false,
		},
		{
			name: "different namespaces",
			deployment: &v1.Deployment{
				Namespace: "default",
			},
			peer: &v1.NetworkPolicyPeer{
				NamespaceSelector: &v1.LabelSelector{
					MatchLabels: map[string]string{
						"key": "value1",
					},
				},
			},
			policyNamespace: "stackrox",
			expected:        false,
		},
		// Todo(cgorman) pod selector and namespace selector combo
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, g.matchPolicyPeer(c.deployment, c.policyNamespace, c.peer))
		})
	}
}

func TestDoesNetworkPolicySelectorApplyToDeployment(t *testing.T) {
	cases := []struct {
		name      string
		d         *v1.Deployment
		np        *v1.NetworkPolicy
		hasPolicy func([]v1.NetworkPolicyType) bool
		expected  bool
	}{
		{
			name: "namespace doesn't match source",
			d: &v1.Deployment{
				Namespace: "default",
			},
			np: &v1.NetworkPolicy{
				Namespace: "stackrox",
			},
			expected: false,
		},
		{
			name: "pod selector doesn't match",
			d: &v1.Deployment{
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key1",
						Value: "value1",
					},
				},
				Namespace: "default",
			},
			np: &v1.NetworkPolicy{
				Namespace: "default",
				Spec: &v1.NetworkPolicySpec{
					PodSelector: &v1.LabelSelector{
						MatchLabels: map[string]string{
							"key1": "value2",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "all matches - has ingress",
			d: &v1.Deployment{
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key1",
						Value: "value1",
					},
				},
				Namespace: "default",
			},
			np: &v1.NetworkPolicy{
				Namespace: "default",
				Spec: &v1.NetworkPolicySpec{
					PodSelector: &v1.LabelSelector{
						MatchLabels: map[string]string{
							"key1": "value1",
						},
					},
				},
			},
			hasPolicy: hasIngress,
			expected:  true,
		},
		{
			name: "all matches - doesn't have egress",
			d: &v1.Deployment{
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key1",
						Value: "value1",
					},
				},
				Namespace: "default",
			},
			np: &v1.NetworkPolicy{
				Namespace: "default",
				Spec: &v1.NetworkPolicySpec{
					PodSelector: &v1.LabelSelector{
						MatchLabels: map[string]string{
							"key1": "value1",
						},
					},
				},
			},
			hasPolicy: hasEgress,
			expected:  false,
		},
		{
			name: "all matches - has egress",
			d: &v1.Deployment{
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key1",
						Value: "value1",
					},
				},
				Namespace: "default",
			},
			np: &v1.NetworkPolicy{
				Namespace: "default",
				Spec: &v1.NetworkPolicySpec{
					PodSelector: &v1.LabelSelector{
						MatchLabels: map[string]string{
							"key1": "value1",
						},
					},
					PolicyTypes: []v1.NetworkPolicyType{v1.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
				},
			},
			hasPolicy: hasEgress,
			expected:  true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, networkPolicySelectorAppliesToDeployment(c.d, c.np, c.hasPolicy))
		})
	}
}

func getExamplePolicy(name string) *v1.NetworkPolicy {
	np, ok := networkPolicyFixtures[name]
	if !ok {
		panic(name)
	}
	return np
}

func egressEdges(src string, dsts ...string) []*v1.NetworkEdge {
	var edges []*v1.NetworkEdge
	for _, d := range dsts {
		edges = append(edges, &v1.NetworkEdge{Source: src, Target: d})
	}
	return edges
}

func ingressEdges(dst string, srcs ...string) []*v1.NetworkEdge {
	var edges []*v1.NetworkEdge
	for _, s := range srcs {
		edges = append(edges, &v1.NetworkEdge{Source: s, Target: dst})
	}
	return edges
}

func fullyConnectedEdges(values ...string) []*v1.NetworkEdge {
	var edges []*v1.NetworkEdge
	for i, value1 := range values {
		for j, value2 := range values {
			if i == j {
				continue
			}
			edges = append(edges, &v1.NetworkEdge{Source: value1, Target: value2})
		}
	}
	return edges
}

func edgeCombiner(edges ...[]*v1.NetworkEdge) []*v1.NetworkEdge {
	var finalEdges []*v1.NetworkEdge
	for _, e := range edges {
		finalEdges = append(finalEdges, e...)
	}
	return finalEdges
}

func createNode(node string, namespace string, policies ...string) *v1.NetworkNode {
	if policies == nil {
		policies = []string{}
	}
	return &v1.NetworkNode{
		Id:        node,
		Namespace: namespace,
		PolicyIds: policies,
	}
}

func deploymentLabels(values ...string) []*v1.Deployment_KeyValue {
	if len(values)%2 != 0 {
		panic("values for deployments labels must be even")
	}
	var keyValues []*v1.Deployment_KeyValue
	for i := 0; i < len(values)/2+1; i += 2 {
		keyValues = append(keyValues, &v1.Deployment_KeyValue{
			Key:   values[i],
			Value: values[i+1],
		})
	}
	return keyValues
}

func TestEvaluateClusters(t *testing.T) {
	g := newMockGraphEvaluator()

	// These are the k8s examples from https://github.com/ahmetb/kubernetes-network-policy-recipes
	// Seems like a good way to verify that the logic is correct
	cases := []struct {
		name        string
		deployments []*v1.Deployment
		nps         []*v1.NetworkPolicy
		edges       []*v1.NetworkEdge
		nodes       []*v1.NetworkNode
	}{
		{
			name: "No policies - fully connected",
			deployments: []*v1.Deployment{
				{
					Id: "d1",
				},
				{
					Id: "d2",
				},
			},
			edges: fullyConnectedEdges("d1", "d2"),
			nodes: []*v1.NetworkNode{
				createNode("d1", ""),
				createNode("d2", ""),
			},
		},
		{
			name: "deny all to app=web",
			deployments: []*v1.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					Labels:    deploymentLabels("app", "web"),
				},
				{
					Id:        "d2",
					Namespace: "default",
				},
				{
					Id:        "d3",
					Namespace: "default",
				},
			},
			edges: edgeCombiner(
				egressEdges("d1", "d2", "d3"),
				fullyConnectedEdges("d2", "d3"),
			),
			nps: []*v1.NetworkPolicy{
				getExamplePolicy("web-deny-all"),
			},
			nodes: []*v1.NetworkNode{
				createNode("d1", "default", "web-deny-all"),
				createNode("d2", "default"),
				createNode("d3", "default"),
			},
		},
		{
			name: "limit traffic to application",
			deployments: []*v1.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					Labels:    deploymentLabels("app", "bookstore", "role", "api"),
				},
				{
					Id:        "d2",
					Namespace: "default",
					Labels:    deploymentLabels("app", "bookstore", "role", "frontend"),
				},
				{
					Id:        "d3",
					Namespace: "default",
					Labels:    deploymentLabels("app", "coffeeshop", "role", "api"),
				},
			},
			edges: edgeCombiner(
				fullyConnectedEdges("d1", "d2"),
				fullyConnectedEdges("d2", "d3"),
				ingressEdges("d3", "d1"),
			),
			nps: []*v1.NetworkPolicy{
				getExamplePolicy("limit-traffic"),
			},
			nodes: []*v1.NetworkNode{
				createNode("d1", "default", "limit-traffic"),
				createNode("d2", "default"),
				createNode("d3", "default"),
			},
		},
		{
			name: "allow all ingress even if deny all",
			deployments: []*v1.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					Labels:    deploymentLabels("app", "web"),
				},
				{
					Id:        "d2",
					Namespace: "default",
				},
				{
					Id:        "d3",
					Namespace: "default",
				},
			},
			edges: edgeCombiner(
				fullyConnectedEdges("d1", "d2", "d3"),
			),
			nps: []*v1.NetworkPolicy{
				getExamplePolicy("web-deny-all"),
				getExamplePolicy("web-allow-all"),
			},
			nodes: []*v1.NetworkNode{
				createNode("d1", "default", "web-allow-all", "web-deny-all"),
				createNode("d2", "default"),
				createNode("d3", "default"),
			},
		},
		{
			name: "DENY all non-whitelisted traffic to a namespace",
			deployments: []*v1.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					Labels:    deploymentLabels("app", "web"),
				},
				{
					Id:        "d2",
					Namespace: "default",
				},
				{
					Id:        "d3",
					Namespace: "stackrox",
				},
			},
			edges: edgeCombiner(
				egressEdges("d1", "d3"),
				egressEdges("d2", "d3"),
			),
			nps: []*v1.NetworkPolicy{
				getExamplePolicy("default-deny-all"),
			},
			nodes: []*v1.NetworkNode{
				createNode("d1", "default", "default-deny-all"),
				createNode("d2", "default", "default-deny-all"),
				createNode("d3", "stackrox"),
			},
		},
		{
			name: "DENY all traffic from other namespaces",
			deployments: []*v1.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
				},
				{
					Id:        "d2",
					Namespace: "default",
				},
				{
					Id:        "d3",
					Namespace: "stackrox",
				},
			},
			edges: edgeCombiner(
				fullyConnectedEdges("d1", "d2"),
				egressEdges("d1", "d3"),
				egressEdges("d2", "d3"),
			),
			nps: []*v1.NetworkPolicy{
				getExamplePolicy("deny-from-other-namespaces"),
			},
			nodes: []*v1.NetworkNode{
				createNode("d1", "default", "deny-from-other-namespaces"),
				createNode("d2", "default", "deny-from-other-namespaces"),
				createNode("d3", "stackrox"),
			},
		},
		{
			name: "Web allow all traffic from other namespaces",
			deployments: []*v1.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					Labels:    deploymentLabels("app", "web"),
				},
				{
					Id:        "d2",
					Namespace: "default",
				},
				{
					Id:        "d3",
					Namespace: "stackrox",
				},
			},
			edges: edgeCombiner(
				fullyConnectedEdges("d1", "d2"),
				fullyConnectedEdges("d1", "d3"),
				egressEdges("d2", "d3"),
			),
			nps: []*v1.NetworkPolicy{
				getExamplePolicy("deny-from-other-namespaces"),
				getExamplePolicy("web-allow-all-namespaces"),
			},
			nodes: []*v1.NetworkNode{
				createNode("d1", "default", "deny-from-other-namespaces", "web-allow-all-namespaces"),
				createNode("d2", "default", "deny-from-other-namespaces"),
				createNode("d3", "stackrox"),
			},
		},
		{
			name: "Web allow all traffic from other namespaces",
			deployments: []*v1.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					Labels:    deploymentLabels("app", "web"),
				},
				{
					Id:        "d2",
					Namespace: "other",
				},
				{
					Id:        "d3",
					Namespace: "stackrox",
				},
			},
			edges: edgeCombiner(
				fullyConnectedEdges("d1", "d3"),
				fullyConnectedEdges("d2", "d3"),
				egressEdges("d1", "d2"),
			),
			nps: []*v1.NetworkPolicy{
				getExamplePolicy("web-allow-stackrox"),
			},
			nodes: []*v1.NetworkNode{
				createNode("d1", "default", "web-allow-stackrox"),
				createNode("d2", "other"),
				createNode("d3", "stackrox"),
			},
		},
		{
			name: "Allow traffic from apps using multiple selectors",
			deployments: []*v1.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					Labels:    deploymentLabels("app", "web", "role", "db"),
				},
				{
					Id:        "d2",
					Namespace: "default",
					Labels:    deploymentLabels("app", "bookstore", "role", "search"),
				},
				{
					Id:        "d3",
					Namespace: "default",
					Labels:    deploymentLabels("app", "bookstore", "role", "api"),
				},
				{
					Id:        "d4",
					Namespace: "default",
				},
			},
			edges: edgeCombiner(
				fullyConnectedEdges("d2", "d3", "d4"),
				egressEdges("d1", "d2", "d3", "d4"),
			),
			nps: []*v1.NetworkPolicy{
				getExamplePolicy("web-deny-all"),
				getExamplePolicy("allow-traffic-from-apps-using-multiple-selectors"),
			},
			nodes: []*v1.NetworkNode{
				createNode("d1", "default", "web-deny-all"),
				createNode("d2", "default"),
				createNode("d3", "default"),
				createNode("d4", "default"),
			},
		},
		{
			name: "web deny egress",
			deployments: []*v1.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					Labels:    deploymentLabels("app", "web"),
				},
				{
					Id:        "d2",
					Namespace: "default",
				},
			},
			edges: edgeCombiner(
				ingressEdges("d1", "d2"),
			),
			nps: []*v1.NetworkPolicy{
				getExamplePolicy("web-deny-egress"),
			},
			nodes: []*v1.NetworkNode{
				createNode("d1", "default", "web-deny-egress"),
				createNode("d2", "default"),
			},
		},
		{
			name: "deny egress from namespace",
			deployments: []*v1.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					Labels:    deploymentLabels("app", "web"),
				},
				{
					Id:        "d2",
					Namespace: "default",
				},
				{
					Id:        "d3",
					Namespace: "stackrox",
				},
			},
			edges: edgeCombiner(
				egressEdges("d3", "d1", "d2"),
			),
			nps: []*v1.NetworkPolicy{
				getExamplePolicy("default-deny-all-egress"),
			},
			nodes: []*v1.NetworkNode{
				createNode("d1", "default", "default-deny-all-egress"),
				createNode("d2", "default", "default-deny-all-egress"),
				createNode("d3", "stackrox"),
			},
		},
		{
			name: "deny external egress from cluster",
			deployments: []*v1.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					Labels:    deploymentLabels("app", "web"),
				},
				{
					Id:        "d2",
					Namespace: "default",
				},
				{
					Id:        "d3",
					Namespace: "stackrox",
				},
			},
			edges: edgeCombiner(
				fullyConnectedEdges("d1", "d2", "d3"),
			),
			nps: []*v1.NetworkPolicy{
				getExamplePolicy("web-deny-external-egress"),
			},
			nodes: []*v1.NetworkNode{
				createNode("d1", "default", "web-deny-external-egress"),
				createNode("d2", "default"),
				createNode("d3", "stackrox"),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			nodes, edges := g.evaluateCluster(c.deployments, c.nps)
			assert.ElementsMatch(t, c.nodes, nodes)
			assert.ElementsMatch(t, c.edges, edges)
		})
	}
}
