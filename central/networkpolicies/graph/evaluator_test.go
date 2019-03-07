package graph

import (
	"bytes"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stretchr/testify/assert"
	k8sV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var networkPolicyFixtures = map[string]*storage.NetworkPolicy{}

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

func (n *namespaceGetter) GetNamespaces() ([]*storage.NamespaceMetadata, error) {
	return []*storage.NamespaceMetadata{
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
		namespace *storage.NamespaceMetadata
		selector  *storage.LabelSelector
		expected  bool
	}{
		{
			name:      "No values in selector - no namespace labels",
			namespace: &storage.NamespaceMetadata{},
			selector:  &storage.LabelSelector{},
			expected:  true,
		},
		{
			name:      "No values in selector - some namespace labels",
			namespace: &storage.NamespaceMetadata{},
			selector:  &storage.LabelSelector{},
			expected:  true,
		},
		{
			name: "matching values in selector",
			namespace: &storage.NamespaceMetadata{
				Labels: map[string]string{
					"hello": "hi",
				},
			},
			selector: &storage.LabelSelector{
				MatchLabels: map[string]string{
					"hello": "hi",
				},
			},
			expected: true,
		},
		{
			name: "non matching values in selector",
			namespace: &storage.NamespaceMetadata{
				Labels: map[string]string{
					"hello": "hi1",
				},
			},
			selector: &storage.LabelSelector{
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
		deployment *storage.Deployment
		selector   *storage.LabelSelector
		expected   bool
	}{
		{
			name:       "No values in selector - no deployment labels",
			deployment: &storage.Deployment{},
			selector:   &storage.LabelSelector{},
			expected:   true,
		},
		{
			name:       "No values in selector - some deployment labels",
			deployment: &storage.Deployment{},
			selector:   &storage.LabelSelector{},
			expected:   true,
		},
		{
			name: "matching values in selector",
			deployment: &storage.Deployment{
				PodLabels: map[string]string{
					"hello": "hi",
				},
			},
			selector: &storage.LabelSelector{
				MatchLabels: map[string]string{
					"hello": "hi",
				},
			},
			expected: true,
		},
		{
			name: "non matching values in selector",
			deployment: &storage.Deployment{
				PodLabels: map[string]string{
					"hello": "hi1",
				},
			},
			selector: &storage.LabelSelector{
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
		policyTypes []storage.NetworkPolicyType
		expected    bool
	}{
		{
			name:        "no values",
			policyTypes: []storage.NetworkPolicyType{},
			expected:    false,
		},
		{
			name:        "ingress only",
			policyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
			expected:    false,
		},
		{
			name:        "egress only",
			policyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
			expected:    true,
		},
		{
			name:        "ingress + egress only",
			policyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
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
		policyTypes []storage.NetworkPolicyType
		expected    bool
	}{
		{
			name:        "no values",
			policyTypes: []storage.NetworkPolicyType{},
			expected:    true,
		},
		{
			name:        "ingress only",
			policyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
			expected:    true,
		},
		{
			name:        "egress only",
			policyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
			expected:    false,
		},
		{
			name:        "ingress + egress only",
			policyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
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
		deployment      *storage.Deployment
		peer            *storage.NetworkPolicyPeer
		policyNamespace string
		expected        bool
	}{
		{
			name:       "ip block",
			deployment: &storage.Deployment{},
			peer:       &storage.NetworkPolicyPeer{IpBlock: &storage.IPBlock{}},
			expected:   false,
		},
		{
			name: "non match pod selector",
			deployment: &storage.Deployment{
				PodLabels: map[string]string{
					"key": "value1",
				},
			},
			peer: &storage.NetworkPolicyPeer{
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{
						"key": "value",
					},
				},
			},
			expected: false,
		},
		{
			name: "match pod selector",
			deployment: &storage.Deployment{
				PodLabels: map[string]string{
					"key": "value",
				},
			},
			peer: &storage.NetworkPolicyPeer{
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{
						"key": "value",
					},
				},
			},
			expected: true,
		},
		{
			name: "match namespace selector",
			deployment: &storage.Deployment{
				Namespace: "default",
			},
			peer: &storage.NetworkPolicyPeer{
				NamespaceSelector: &storage.LabelSelector{
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
			deployment: &storage.Deployment{
				Namespace: "default",
			},
			peer: &storage.NetworkPolicyPeer{
				NamespaceSelector: &storage.LabelSelector{
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
			deployment: &storage.Deployment{
				Namespace: "default",
			},
			peer: &storage.NetworkPolicyPeer{
				NamespaceSelector: &storage.LabelSelector{
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
			assert.Equal(t, c.expected, g.deploymentMatcher.matchPolicyPeer(c.deployment, c.policyNamespace, c.peer))
		})
	}
}

func TestIngressNetworkPolicySelectorAppliesToDeployment(t *testing.T) {
	cases := []struct {
		name     string
		d        *storage.Deployment
		np       *storage.NetworkPolicy
		expected bool
	}{
		{
			name: "namespace doesn't match source",
			d: &storage.Deployment{
				Namespace: "default",
			},
			np: &storage.NetworkPolicy{
				Namespace: "stackrox",
			},
			expected: false,
		},
		{
			name: "pod selector doesn't match",
			d: &storage.Deployment{
				PodLabels: map[string]string{
					"key1": "value1",
				},
				Namespace: "default",
			},
			np: &storage.NetworkPolicy{
				Namespace: "default",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
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
			d: &storage.Deployment{
				PodLabels: map[string]string{
					"key1": "value1",
				},
				Namespace: "default",
			},
			np: &storage.NetworkPolicy{
				Namespace: "default",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
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
			d: &storage.Deployment{
				PodLabels: map[string]string{
					"key1": "value1",
				},
				Namespace: "default",
			},
			np: &storage.NetworkPolicy{
				Namespace: "default",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{
							"key1": "value1",
						},
					},
					PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
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
		d              *storage.Deployment
		np             *storage.NetworkPolicy
		expected       bool
		internetAccess bool
	}{
		{
			name: "namespace doesn't match source",
			d: &storage.Deployment{
				Namespace: "default",
			},
			np: &storage.NetworkPolicy{
				Namespace: "stackrox",
			},
			expected: false,
		},
		{
			name: "pod selector doesn't match",
			d: &storage.Deployment{
				PodLabels: map[string]string{
					"key1": "value1",
				},
				Namespace: "default",
			},
			np: &storage.NetworkPolicy{
				Namespace: "default",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
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
			d: &storage.Deployment{
				PodLabels: map[string]string{
					"key1": "value1",
				},
				Namespace: "default",
			},
			np: &storage.NetworkPolicy{
				Namespace: "default",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
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
			d: &storage.Deployment{
				PodLabels: map[string]string{
					"key1": "value1",
				},
				Namespace: "default",
			},
			np: &storage.NetworkPolicy{
				Namespace: "default",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{
							"key1": "value1",
						},
					},
					PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
				},
			},
			expected: true,
		},
		{
			name: "all matches - has egress and ip block",
			d: &storage.Deployment{
				PodLabels: map[string]string{
					"key1": "value1",
				},
				Namespace: "default",
			},
			np: &storage.NetworkPolicy{
				Namespace: "default",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{
							"key1": "value1",
						},
					},
					Egress: []*storage.NetworkPolicyEgressRule{
						{
							To: []*storage.NetworkPolicyPeer{
								{
									IpBlock: &storage.IPBlock{
										Cidr: "127.0.0.1/32",
									},
								},
							},
						},
					},
					PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
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

func getExamplePolicy(name string) *storage.NetworkPolicy {
	np, ok := networkPolicyFixtures[name]
	if !ok {
		panic(name)
	}
	return np
}

type edge struct {
	Source, Target string
}

func egressEdges(src string, dsts ...string) []edge {
	var edges []edge
	for _, d := range dsts {
		edges = append(edges, edge{Source: src, Target: d})
	}
	return edges
}

func ingressEdges(dst string, srcs ...string) []edge {
	var edges []edge
	for _, s := range srcs {
		edges = append(edges, edge{Source: s, Target: dst})
	}
	return edges
}

func fullyConnectedEdges(values ...string) []edge {
	var edges []edge
	for i, value1 := range values {
		for j, value2 := range values {
			if i == j {
				continue
			}
			edges = append(edges, edge{Source: value1, Target: value2})
		}
	}
	return edges
}

func flattenEdges(edges ...[]edge) []edge {
	var finalEdges []edge
	for _, e := range edges {
		finalEdges = append(finalEdges, e...)
	}
	return finalEdges
}

func mockNode(node string, namespace string, internetAccess bool, policies ...string) *v1.NetworkNode {
	if policies == nil {
		policies = []string{}
	}
	return &v1.NetworkNode{
		Entity: &storage.NetworkEntityInfo{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			Id:   node,
			Desc: &storage.NetworkEntityInfo_Deployment_{
				Deployment: &storage.NetworkEntityInfo_Deployment{
					Namespace: namespace,
				},
			},
		},
		PolicyIds:      policies,
		InternetAccess: internetAccess,
		OutEdges:       make(map[int32]*v1.NetworkEdgePropertiesBundle),
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
		deployments []*storage.Deployment
		nps         []*storage.NetworkPolicy
		edges       []edge
		nodes       []*v1.NetworkNode
	}{
		{
			name: "No policies - fully connected",
			deployments: []*storage.Deployment{
				{
					Id: "d1",
				},
				{
					Id: "d2",
				},
			},
			edges: fullyConnectedEdges("d1", "d2"),
			nodes: []*v1.NetworkNode{
				mockNode("d1", "", true),
				mockNode("d2", "", true),
			},
		},
		{
			name: "deny all to app=web",
			deployments: []*storage.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "web"),
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
			edges: flattenEdges(
				egressEdges("d1", "d2", "d3"),
				fullyConnectedEdges("d2", "d3"),
			),
			nps: []*storage.NetworkPolicy{
				getExamplePolicy("web-deny-all"),
			},
			nodes: []*v1.NetworkNode{
				mockNode("d1", "default", true, "web-deny-all"),
				mockNode("d2", "default", true),
				mockNode("d3", "default", true),
			},
		},
		{
			name: "limit traffic to application",
			deployments: []*storage.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "bookstore", "role", "api"),
				},
				{
					Id:        "d2",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "bookstore", "role", "frontend"),
				},
				{
					Id:        "d3",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "coffeeshop", "role", "api"),
				},
			},
			edges: flattenEdges(
				fullyConnectedEdges("d1", "d2"),
				fullyConnectedEdges("d2", "d3"),
				ingressEdges("d3", "d1"),
			),
			nps: []*storage.NetworkPolicy{
				getExamplePolicy("limit-traffic"),
			},
			nodes: []*v1.NetworkNode{
				mockNode("d1", "default", true, "limit-traffic"),
				mockNode("d2", "default", true),
				mockNode("d3", "default", true),
			},
		},
		{
			name: "allow all ingress even if deny all",
			deployments: []*storage.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "web"),
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
			edges: flattenEdges(
				fullyConnectedEdges("d1", "d2", "d3"),
			),
			nps: []*storage.NetworkPolicy{
				getExamplePolicy("web-deny-all"),
				getExamplePolicy("web-allow-all"),
			},
			nodes: []*v1.NetworkNode{
				mockNode("d1", "default", true, "web-allow-all", "web-deny-all"),
				mockNode("d2", "default", true),
				mockNode("d3", "default", true),
			},
		},
		{
			name: "DENY all non-whitelisted traffic to a namespace",
			deployments: []*storage.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "web"),
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
			edges: flattenEdges(
				egressEdges("d1", "d3"),
				egressEdges("d2", "d3"),
			),
			nps: []*storage.NetworkPolicy{
				getExamplePolicy("default-deny-all"),
			},
			nodes: []*v1.NetworkNode{
				mockNode("d1", "default", true, "default-deny-all"),
				mockNode("d2", "default", true, "default-deny-all"),
				mockNode("d3", "stackrox", true),
			},
		},
		{
			name: "DENY all traffic from other namespaces",
			deployments: []*storage.Deployment{
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
			edges: flattenEdges(
				fullyConnectedEdges("d1", "d2"),
				egressEdges("d1", "d3"),
				egressEdges("d2", "d3"),
			),
			nps: []*storage.NetworkPolicy{
				getExamplePolicy("deny-from-other-namespaces"),
			},
			nodes: []*v1.NetworkNode{
				mockNode("d1", "default", true, "deny-from-other-namespaces"),
				mockNode("d2", "default", true, "deny-from-other-namespaces"),
				mockNode("d3", "stackrox", true),
			},
		},
		{
			name: "Web allow all traffic from other namespaces",
			deployments: []*storage.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "web"),
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
			edges: flattenEdges(
				fullyConnectedEdges("d1", "d2"),
				fullyConnectedEdges("d1", "d3"),
				egressEdges("d2", "d3"),
			),
			nps: []*storage.NetworkPolicy{
				getExamplePolicy("deny-from-other-namespaces"),
				getExamplePolicy("web-allow-all-namespaces"),
			},
			nodes: []*v1.NetworkNode{
				mockNode("d1", "default", true, "deny-from-other-namespaces", "web-allow-all-namespaces"),
				mockNode("d2", "default", true, "deny-from-other-namespaces"),
				mockNode("d3", "stackrox", true),
			},
		},
		{
			name: "Web allow all traffic from other namespaces",
			deployments: []*storage.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "web"),
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
			edges: flattenEdges(
				fullyConnectedEdges("d1", "d3"),
				fullyConnectedEdges("d2", "d3"),
				egressEdges("d1", "d2"),
			),
			nps: []*storage.NetworkPolicy{
				getExamplePolicy("web-allow-stackrox"),
			},
			nodes: []*v1.NetworkNode{
				mockNode("d1", "default", true, "web-allow-stackrox"),
				mockNode("d2", "other", true),
				mockNode("d3", "stackrox", true),
			},
		},
		{
			name: "Allow traffic from apps using multiple selectors",
			deployments: []*storage.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "web", "role", "db"),
				},
				{
					Id:        "d2",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "bookstore", "role", "search"),
				},
				{
					Id:        "d3",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "bookstore", "role", "api"),
				},
				{
					Id:        "d4",
					Namespace: "default",
				},
			},
			edges: flattenEdges(
				fullyConnectedEdges("d2", "d3", "d4"),
				egressEdges("d1", "d2", "d3", "d4"),
			),
			nps: []*storage.NetworkPolicy{
				getExamplePolicy("web-deny-all"),
				getExamplePolicy("allow-traffic-from-apps-using-multiple-selectors"),
			},
			nodes: []*v1.NetworkNode{
				mockNode("d1", "default", true, "web-deny-all"),
				mockNode("d2", "default", true),
				mockNode("d3", "default", true),
				mockNode("d4", "default", true),
			},
		},
		{
			name: "web deny egress",
			deployments: []*storage.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "web"),
				},
				{
					Id:        "d2",
					Namespace: "default",
				},
			},
			edges: flattenEdges(
				ingressEdges("d1", "d2"),
			),
			nps: []*storage.NetworkPolicy{
				getExamplePolicy("web-deny-egress"),
			},
			nodes: []*v1.NetworkNode{
				mockNode("d1", "default", false, "web-deny-egress"),
				mockNode("d2", "default", true),
			},
		},
		{
			name: "deny egress from namespace",
			deployments: []*storage.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "web"),
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
			edges: flattenEdges(
				egressEdges("d3", "d1", "d2"),
			),
			nps: []*storage.NetworkPolicy{
				getExamplePolicy("default-deny-all-egress"),
			},
			nodes: []*v1.NetworkNode{
				mockNode("d1", "default", false, "default-deny-all-egress"),
				mockNode("d2", "default", false, "default-deny-all-egress"),
				mockNode("d3", "stackrox", true),
			},
		},
		{
			name: "deny internetAccess egress from cluster",
			deployments: []*storage.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "web"),
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
			edges: flattenEdges(
				fullyConnectedEdges("d1", "d2", "d3"),
			),
			nps: []*storage.NetworkPolicy{
				getExamplePolicy("web-deny-external-egress"),
			},
			nodes: []*v1.NetworkNode{
				mockNode("d1", "default", false, "web-deny-external-egress"),
				mockNode("d2", "default", true),
				mockNode("d3", "stackrox", true),
			},
		},
		{
			name: "deny all ingress except for app = web",
			deployments: []*storage.Deployment{
				{
					Id:        "d1",
					Namespace: "qa",
					PodLabels: deploymentLabels("app", "web"),
				},
				{
					Id:        "d2",
					Namespace: "qa",
					PodLabels: deploymentLabels("app", "client"),
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
			edges: flattenEdges(
				ingressEdges("d1", "d2", "d3", "d4"),
				ingressEdges("d3", "d1", "d2", "d4"),
				ingressEdges("d4", "d1", "d2", "d3"),
			),
			nps: []*storage.NetworkPolicy{
				getExamplePolicy("deny-all-ingress"),
				getExamplePolicy("allow-ingress-to-web"),
			},
			nodes: []*v1.NetworkNode{
				mockNode("d1", "qa", true, "allow-ingress-to-web", "deny-all-ingress"),
				mockNode("d2", "qa", true, "deny-all-ingress"),
				mockNode("d3", "stackrox", true),
				mockNode("d4", "default", true),
			},
		},
	}
	for _, c := range cases {
		populateOutEdges(c.nodes, c.edges)
		t.Run(c.name, func(t *testing.T) {
			nodes := g.evaluate(c.deployments, c.nps)
			assert.ElementsMatch(t, c.nodes, nodes)
		})
	}
}

func TestGetApplicable(t *testing.T) {
	g := newMockGraphEvaluator()

	// These are the k8s examples from https://github.com/ahmetb/kubernetes-network-policy-recipes
	// Seems like a good way to verify that the logic is correct
	cases := []struct {
		name        string
		deployments []*storage.Deployment
		policies    []*storage.NetworkPolicy
		expected    []*storage.NetworkPolicy
	}{
		{
			name: "No policies",
			deployments: []*storage.Deployment{
				{
					Id: "d1",
				},
				{
					Id: "d2",
				},
			},
		},
		{
			name: "deny all to app=web with match",
			deployments: []*storage.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "web"),
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
			policies: []*storage.NetworkPolicy{
				getExamplePolicy("web-deny-all"),
			},
			expected: []*storage.NetworkPolicy{
				getExamplePolicy("web-deny-all"),
			},
		},
		{
			name: "limit traffic to application with match",
			deployments: []*storage.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "bookstore", "role", "api"),
				},
				{
					Id:        "d2",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "bookstore", "role", "frontend"),
				},
				{
					Id:        "d3",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "coffeeshop", "role", "api"),
				},
			},
			policies: []*storage.NetworkPolicy{
				getExamplePolicy("limit-traffic"),
			},
			expected: []*storage.NetworkPolicy{
				getExamplePolicy("limit-traffic"),
			},
		},
		{
			name: "limit traffic to application no match",
			deployments: []*storage.Deployment{
				{
					Id:        "d1",
					Namespace: "default",
					PodLabels: deploymentLabels("app", "web"),
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
			policies: []*storage.NetworkPolicy{
				getExamplePolicy("limit-traffic"),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := g.GetAppliedPolicies(c.deployments, c.policies)
			assert.ElementsMatch(t, c.expected, actual)
		})
	}
}

func populateOutEdges(nodes []*v1.NetworkNode, edges []edge) {
	indexMap := make(map[string]int)
	for i, node := range nodes {
		indexMap[node.Entity.Id] = i
	}

	for _, e := range edges {
		if e.Source == e.Target {
			continue
		}
		srcIndex := indexMap[e.Source]
		srcNode := nodes[srcIndex]
		tgtIndex := indexMap[e.Target]
		srcNode.OutEdges[int32(tgtIndex)] = &v1.NetworkEdgePropertiesBundle{}
	}
}
