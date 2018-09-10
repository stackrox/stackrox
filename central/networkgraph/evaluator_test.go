package networkgraph

import (
	"bytes"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
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
		np := networkPolicyConversion.KubernetesNetworkPolicyWrap{NetworkPolicy: &k8sNp}.ToRoxNetworkPolicy()
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

func newMockGraphEvaluator() *evaluatorImpl {
	return newGraphEvaluator(&namespaceGetter{})
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
			deployment: &v1.Deployment{
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

				Labels: map[string]string{
					"key": "value1",
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
				Labels: map[string]string{
					"key": "value",
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

func TestIngressNetworkPolicySelectorAppliesToDeployment(t *testing.T) {
	cases := []struct {
		name     string
		d        *v1.Deployment
		np       *v1.NetworkPolicy
		expected bool
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
				Labels: map[string]string{
					"key1": "value1",
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
				Labels: map[string]string{
					"key1": "value1",
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
			expected: true,
		},
		{
			name: "all matches - doesn't have ingress",
			d: &v1.Deployment{
				Labels: map[string]string{
					"key1": "value1",
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
			expected: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, ingressNetworkPolicySelectorAppliesToDeployment(c.d, c.np))
		})
	}
}

func TestEgressNetworkPolicySelectorAppliesToDeployment(t *testing.T) {
	cases := []struct {
		name           string
		d              *v1.Deployment
		np             *v1.NetworkPolicy
		expected       bool
		internetAccess bool
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
				Labels: map[string]string{
					"key1": "value1",
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
			name: "all matches - doesn't have egress",
			d: &v1.Deployment{
				Labels: map[string]string{
					"key1": "value1",
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
			expected: false,
		},
		{
			name: "all matches - has egress",
			d: &v1.Deployment{
				Labels: map[string]string{
					"key1": "value1",
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
			expected: true,
		},
		{
			name: "all matches - has egress and ip block",
			d: &v1.Deployment{
				Labels: map[string]string{
					"key1": "value1",
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
					Egress: []*v1.NetworkPolicyEgressRule{
						{
							To: []*v1.NetworkPolicyPeer{
								{
									IpBlock: &v1.IPBlock{
										Cidr: "127.0.0.1/32",
									},
								},
							},
						},
					},
					PolicyTypes: []v1.NetworkPolicyType{v1.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
				},
			},
			expected:       true,
			internetAccess: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			matches, internetAccess := egressNetworkPolicySelectorAppliesToDeployment(c.d, c.np)
			assert.Equal(t, c.expected, matches)
			assert.Equal(t, c.internetAccess, internetAccess)
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

func createNode(node string, namespace string, internetAccess bool, policies ...string) *v1.NetworkNode {
	if policies == nil {
		policies = []string{}
	}
	return &v1.NetworkNode{
		Id:             node,
		Namespace:      namespace,
		PolicyIds:      policies,
		InternetAccess: internetAccess,
	}
}

func deploymentLabels(values ...string) map[string]string {
	if len(values)%2 != 0 {
		panic("values for deployments labels must be even")
	}
	m := make(map[string]string)
	for i := 0; i < len(values)/2+1; i += 2 {
		m[values[i]] = values[i+1]
	}
	return m
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
				createNode("d1", "", true),
				createNode("d2", "", true),
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
				createNode("d1", "default", true, "web-deny-all"),
				createNode("d2", "default", true),
				createNode("d3", "default", true),
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
				createNode("d1", "default", true, "limit-traffic"),
				createNode("d2", "default", true),
				createNode("d3", "default", true),
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
				createNode("d1", "default", true, "web-allow-all", "web-deny-all"),
				createNode("d2", "default", true),
				createNode("d3", "default", true),
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
				createNode("d1", "default", true, "default-deny-all"),
				createNode("d2", "default", true, "default-deny-all"),
				createNode("d3", "stackrox", true),
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
				createNode("d1", "default", true, "deny-from-other-namespaces"),
				createNode("d2", "default", true, "deny-from-other-namespaces"),
				createNode("d3", "stackrox", true),
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
				createNode("d1", "default", true, "deny-from-other-namespaces", "web-allow-all-namespaces"),
				createNode("d2", "default", true, "deny-from-other-namespaces"),
				createNode("d3", "stackrox", true),
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
				createNode("d1", "default", true, "web-allow-stackrox"),
				createNode("d2", "other", true),
				createNode("d3", "stackrox", true),
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
				createNode("d1", "default", true, "web-deny-all"),
				createNode("d2", "default", true),
				createNode("d3", "default", true),
				createNode("d4", "default", true),
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
				createNode("d1", "default", false, "web-deny-egress"),
				createNode("d2", "default", true),
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
				createNode("d1", "default", false, "default-deny-all-egress"),
				createNode("d2", "default", false, "default-deny-all-egress"),
				createNode("d3", "stackrox", true),
			},
		},
		{
			name: "deny internetAccess egress from cluster",
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
				createNode("d1", "default", false, "web-deny-external-egress"),
				createNode("d2", "default", true),
				createNode("d3", "stackrox", true),
			},
		},
		{
			name: "deny all ingress except for app = web",
			deployments: []*v1.Deployment{
				{
					Id:        "d1",
					Namespace: "qa",
					Labels:    deploymentLabels("app", "web"),
				},
				{
					Id:        "d2",
					Namespace: "qa",
					Labels:    deploymentLabels("app", "client"),
				},
				{
					Id:        "d3",
					Namespace: "stackrox",
				},
				{
					Id:        "d4",
					Namespace: "default",
				},
			},
			edges: edgeCombiner(
				ingressEdges("d1", "d2", "d3", "d4"),
				ingressEdges("d3", "d1", "d2", "d4"),
				ingressEdges("d4", "d1", "d2", "d3"),
			),
			nps: []*v1.NetworkPolicy{
				getExamplePolicy("deny-all-ingress"),
				getExamplePolicy("allow-ingress-to-web"),
			},
			nodes: []*v1.NetworkNode{
				createNode("d1", "qa", true, "allow-ingress-to-web", "deny-all-ingress"),
				createNode("d2", "qa", true, "deny-all-ingress"),
				createNode("d3", "stackrox", true),
				createNode("d4", "default", true),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			nodes, edges := g.evaluate(c.deployments, c.nps)
			assert.ElementsMatch(t, c.nodes, nodes)
			assert.ElementsMatch(t, c.edges, edges)
		})
	}
}
