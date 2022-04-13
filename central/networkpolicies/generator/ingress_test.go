package generator

import (
	"sort"
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/namespaces"
	"github.com/stackrox/stackrox/pkg/networkgraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	httpsPort = portDesc{
		l4proto: storage.L4Protocol_L4_PROTOCOL_TCP,
		port:    443,
	}

	dnsPort = portDesc{
		l4proto: storage.L4Protocol_L4_PROTOCOL_UDP,
		port:    53,
	}
)

func makeNumPort(numPort int32) *storage.NetworkPolicyPort_Port {
	return &storage.NetworkPolicyPort_Port{
		Port: numPort,
	}
}

func createDeploymentNode(id, name, namespace string, selectorLabels map[string]string) *node {
	return &node{
		entity: networkgraph.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   id,
		},
		deployment: &storage.Deployment{
			Id:        id,
			Name:      name,
			Namespace: namespace,
			LabelSelector: &storage.LabelSelector{
				MatchLabels: selectorLabels,
			},
		},
		incoming: make(map[portDesc]*ingressInfo),
		outgoing: make(map[portDesc]peers),
	}
}

func TestGenerateIngressRule_WithInternetIngress(t *testing.T) {
	t.Parallel()

	internetNode := &node{
		entity: networkgraph.Entity{
			Type: storage.NetworkEntityInfo_INTERNET,
		},
	}

	nss := map[string]*storage.NamespaceMetadata{
		"ns": {
			Id:   "ns",
			Name: "ns",
			Labels: map[string]string{
				namespaces.NamespaceNameLabel: "ns",
			},
		},
	}

	deployment0 := createDeploymentNode("deployment0", "deployment0", "ns", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "deployment1", "ns", nil)
	deployment1.incoming[portDesc{}] = &ingressInfo{
		peers: peers{
			deployment0:  struct{}{},
			internetNode: struct{}{},
		},
	}

	rule := generateIngressRule(deployment1, portDesc{}, nss)
	assert.Equal(t, allowAllIngress, rule)
}

func TestGenerateIngressRule_WithInternetIngress_WithPorts(t *testing.T) {
	t.Parallel()

	internetNode := &node{
		entity: networkgraph.Entity{
			Type: storage.NetworkEntityInfo_INTERNET,
		},
	}

	nss := map[string]*storage.NamespaceMetadata{
		"ns": {
			Id:   "ns",
			Name: "ns",
			Labels: map[string]string{
				namespaces.NamespaceNameLabel: "ns",
			},
		},
	}

	deployment0 := createDeploymentNode("deployment0", "deployment0", "ns", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "deployment1", "ns", nil)
	deployment1.incoming[httpsPort] = &ingressInfo{
		peers: peers{
			deployment0:  struct{}{},
			internetNode: struct{}{},
		},
	}
	deployment1.incoming[dnsPort] = &ingressInfo{
		peers: peers{
			deployment0: struct{}{},
		},
	}

	expectedRules := []*storage.NetworkPolicyIngressRule{
		{
			Ports: []*storage.NetworkPolicyPort{
				{
					Protocol: storage.Protocol_TCP_PROTOCOL,
					PortRef:  makeNumPort(443),
				},
			},
			From: allowAllIngress.GetFrom(),
		},
		{
			Ports: []*storage.NetworkPolicyPort{
				{
					Protocol: storage.Protocol_UDP_PROTOCOL,
					PortRef:  makeNumPort(53),
				},
			},
			From: []*storage.NetworkPolicyPeer{
				{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{
							"app": "foo",
						},
					},
				},
			},
		},
	}

	rules := generateIngressRules(deployment1, nss)
	assert.ElementsMatch(t, expectedRules, rules)
}

func TestGenerateIngressRule_WithInternetExposure(t *testing.T) {
	t.Parallel()

	deployment0 := createDeploymentNode("deployment0", "deployment0", "ns", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "deployment1", "ns", nil)

	deployment1.deployment.Ports = []*storage.PortConfig{
		{
			ContainerPort: 443,
			Exposure:      storage.PortConfig_EXTERNAL,
		},
	}
	deployment1.incoming[portDesc{}] = &ingressInfo{
		peers: peers{
			deployment0: struct{}{},
		},
	}
	deployment1.populateExposureInfo(false)

	nss := map[string]*storage.NamespaceMetadata{
		"ns": {
			Id:   "ns",
			Name: "ns",
			Labels: map[string]string{
				namespaces.NamespaceNameLabel: "ns",
			},
		},
	}

	rule := generateIngressRule(deployment1, portDesc{}, nss)
	assert.Equal(t, allowAllIngress, rule)
}

func TestGenerateIngressRule_WithInternetExposure_WithPorts(t *testing.T) {
	t.Parallel()

	deployment0 := createDeploymentNode("deployment0", "deployment0", "ns", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "deployment1", "ns", nil)

	deployment1.deployment.Ports = []*storage.PortConfig{
		{
			ContainerPort: 443,
			Protocol:      "TCP",
			Exposure:      storage.PortConfig_EXTERNAL,
		},
		{
			ContainerPort: 80,
			Protocol:      "TCP",
			Exposure:      storage.PortConfig_NODE,
		},
	}
	deployment1.incoming[httpsPort] = &ingressInfo{
		peers: peers{
			deployment0: struct{}{},
		},
	}
	deployment1.incoming[dnsPort] = &ingressInfo{
		peers: peers{
			deployment0: struct{}{},
		},
	}
	deployment1.populateExposureInfo(true)

	nss := map[string]*storage.NamespaceMetadata{
		"ns": {
			Id:   "ns",
			Name: "ns",
			Labels: map[string]string{
				namespaces.NamespaceNameLabel: "ns",
			},
		},
	}

	expectedRules := []*storage.NetworkPolicyIngressRule{
		{
			Ports: []*storage.NetworkPolicyPort{
				{
					Protocol: storage.Protocol_TCP_PROTOCOL,
					PortRef:  makeNumPort(443),
				},
			},
			From: allowAllIngress.GetFrom(),
		},
		{
			Ports: []*storage.NetworkPolicyPort{
				{
					Protocol: storage.Protocol_TCP_PROTOCOL,
					PortRef:  makeNumPort(80),
				},
			},
			From: allowAllIngress.GetFrom(),
		},
		{
			Ports: []*storage.NetworkPolicyPort{
				{
					Protocol: storage.Protocol_UDP_PROTOCOL,
					PortRef:  makeNumPort(53),
				},
			},
			From: []*storage.NetworkPolicyPeer{
				{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{
							"app": "foo",
						},
					},
				},
			},
		},
	}

	rules := generateIngressRules(deployment1, nss)
	assert.ElementsMatch(t, expectedRules, rules)
}

func TestGenerateIngressRule_WithoutInternet(t *testing.T) {
	t.Parallel()

	deployment0 := createDeploymentNode("deployment0", "deployment0", "ns1", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "deployment1", "ns2", map[string]string{"app": "bar"})

	tgtDeployment := createDeploymentNode("tgtDeployment", "tgtDeployment", "ns1", nil)
	tgtDeployment.incoming[portDesc{}] = &ingressInfo{
		peers: peers{
			deployment0: struct{}{},
			deployment1: struct{}{},
		},
	}

	nss := map[string]*storage.NamespaceMetadata{
		"ns1": {
			Id:   "ns1",
			Name: "ns1",
			Labels: map[string]string{
				namespaces.NamespaceNameLabel: "ns1",
			},
		},
		"ns2": {
			Id:   "ns2",
			Name: "ns2",
			Labels: map[string]string{
				namespaces.NamespaceNameLabel: "ns2",
			},
		},
	}

	expectedPeers := []*storage.NetworkPolicyPeer{
		{
			PodSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{"app": "foo"},
			},
		},
		{
			NamespaceSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{namespaces.NamespaceNameLabel: "ns2"},
			},
			PodSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{"app": "bar"},
			},
		},
	}

	rule := generateIngressRule(tgtDeployment, portDesc{}, nss)
	assert.ElementsMatch(t, expectedPeers, rule.From)
}

func TestGenerateIngressRule_WithoutInternet_WithPorts(t *testing.T) {
	t.Parallel()

	deployment0 := createDeploymentNode("deployment0", "deployment0", "ns1", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "deployment1", "ns2", map[string]string{"app": "bar"})

	tgtDeployment := createDeploymentNode("tgtDeployment", "tgtDeployment", "ns1", nil)
	tgtDeployment.incoming[httpsPort] = &ingressInfo{
		peers: peers{
			deployment0: struct{}{},
			deployment1: struct{}{},
		},
	}
	tgtDeployment.incoming[dnsPort] = &ingressInfo{
		peers: peers{
			deployment0: struct{}{},
		},
	}

	nss := map[string]*storage.NamespaceMetadata{
		"ns1": {
			Id:   "ns1",
			Name: "ns1",
			Labels: map[string]string{
				namespaces.NamespaceNameLabel: "ns1",
			},
		},
		"ns2": {
			Id:   "ns2",
			Name: "ns2",
			Labels: map[string]string{
				namespaces.NamespaceNameLabel: "ns2",
			},
		},
	}

	expectedRules := []*storage.NetworkPolicyIngressRule{
		{
			Ports: []*storage.NetworkPolicyPort{
				{
					Protocol: storage.Protocol_TCP_PROTOCOL,
					PortRef:  makeNumPort(443),
				},
			},
			From: []*storage.NetworkPolicyPeer{
				{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{"app": "foo"},
					},
				},
				{
					NamespaceSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{namespaces.NamespaceNameLabel: "ns2"},
					},
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{"app": "bar"},
					},
				},
			},
		},
		{
			Ports: []*storage.NetworkPolicyPort{
				{
					Protocol: storage.Protocol_UDP_PROTOCOL,
					PortRef:  makeNumPort(53),
				},
			},
			From: []*storage.NetworkPolicyPeer{
				{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{"app": "foo"},
					},
				},
			},
		},
	}

	rules := generateIngressRules(tgtDeployment, nss)
	// Modify rules to make sure that all namespace-local peers appear before all foreign-namespace-peers;
	// otherwise the comparison below might fail.
	for _, rule := range rules {
		sort.SliceStable(rule.From, func(i, j int) bool {
			return rule.From[i].NamespaceSelector == nil && rule.From[j].NamespaceSelector != nil
		})
	}
	assert.ElementsMatch(t, expectedRules, rules)
}

func TestGenerateIngressRule_ScopeAlienDeployment(t *testing.T) {
	t.Parallel()

	deployment0 := createDeploymentNode("deployment0", "deployment0", "ns1", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "deployment1", "ns2", map[string]string{"app": "bar"})
	deployment1.masked = true
	tgtDeployment := createDeploymentNode("tgtDeployment", "tgtDeployment", "ns1", nil)
	tgtDeployment.incoming[portDesc{}] = &ingressInfo{
		peers: peers{
			deployment0: struct{}{},
			deployment1: struct{}{},
		},
	}

	nss := map[string]*storage.NamespaceMetadata{
		"ns": {
			Id:   "ns1",
			Name: "ns1",
			Labels: map[string]string{
				namespaces.NamespaceNameLabel: "ns1",
			},
		},
	}

	expectedPeers := []*storage.NetworkPolicyPeer{
		{
			NamespaceSelector: &storage.LabelSelector{},
			PodSelector:       &storage.LabelSelector{},
		},
	}
	rule := generateIngressRule(tgtDeployment, portDesc{}, nss)
	assert.Equal(t, expectedPeers, rule.From)
}

func TestGenerateIngressRule_ScopeAlienNSOnly(t *testing.T) {
	t.Parallel()

	deployment0 := createDeploymentNode("deployment0", "deployment0", "ns1", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "deployment1", "ns2", map[string]string{"app": "bar"})
	tgtDeployment := createDeploymentNode("tgtDeployment", "tgtDeployment", "ns1", nil)
	tgtDeployment.incoming[portDesc{}] = &ingressInfo{
		peers: peers{
			deployment0: struct{}{},
			deployment1: struct{}{},
		},
	}

	nss := map[string]*storage.NamespaceMetadata{
		"ns": {
			Id:   "ns1",
			Name: "ns1",
			Labels: map[string]string{
				namespaces.NamespaceNameLabel: "ns1",
			},
		},
	}

	expectedPeers := []*storage.NetworkPolicyPeer{
		{
			PodSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{"app": "foo"},
			},
		},
		{
			NamespaceSelector: &storage.LabelSelector{},
			PodSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{"app": "bar"},
			},
		},
	}
	rule := generateIngressRule(tgtDeployment, portDesc{}, nss)
	assert.ElementsMatch(t, expectedPeers, rule.From)
}

func TestGenerateIngressRule_FromProtectedNS(t *testing.T) {
	t.Parallel()

	tgtDeployment := createDeploymentNode("tgtDeployment", "tgtDeployment", "ns1", nil)

	deployment0 := createDeploymentNode("deployment0", "deployment0", "kube-system", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "deployment1", "ns2", map[string]string{"app": "bar"})

	tgtDeployment.incoming[portDesc{}] = &ingressInfo{
		peers: peers{
			deployment0: struct{}{},
			deployment1: struct{}{},
		},
	}

	nss := map[string]*storage.NamespaceMetadata{
		"ns1": {
			Id:   "ns1",
			Name: "ns1",
			Labels: map[string]string{
				namespaces.NamespaceNameLabel: "ns1",
			},
		},
		"ns2": {
			Id:   "ns2",
			Name: "ns2",
			Labels: map[string]string{
				namespaces.NamespaceNameLabel: "ns2",
			},
		},
		"kube-system": {
			Id:   "kube-system",
			Name: "kube-system",
			Labels: map[string]string{
				namespaces.NamespaceNameLabel: "kube-system",
			},
		},
	}

	expectedPeers := []*storage.NetworkPolicyPeer{
		{
			NamespaceSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{
					namespaces.NamespaceNameLabel: "kube-system",
				},
			},
			PodSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{"app": "foo"},
			},
		},
		{
			NamespaceSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{
					namespaces.NamespaceNameLabel: "ns2",
				},
			},
			PodSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{"app": "bar"},
			},
		},
	}

	rules := generateIngressRules(tgtDeployment, nss)
	require.Len(t, rules, 1)
	assert.ElementsMatch(t, expectedPeers, rules[0].From)
}
