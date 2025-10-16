package generator

import (
	"sort"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
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
	ls := &storage.LabelSelector{}
	ls.SetMatchLabels(selectorLabels)
	deployment := &storage.Deployment{}
	deployment.SetId(id)
	deployment.SetName(name)
	deployment.SetNamespace(namespace)
	deployment.SetLabelSelector(ls)
	return &node{
		entity: networkgraph.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   id,
		},
		deployment: deployment,
		incoming:   make(map[portDesc]*ingressInfo),
		outgoing:   make(map[portDesc]peers),
	}
}

func TestGenerateIngressRule_WithInternetIngress(t *testing.T) {

	internetNode := &node{
		entity: networkgraph.Entity{
			Type: storage.NetworkEntityInfo_INTERNET,
		},
	}

	nm := &storage.NamespaceMetadata{}
	nm.SetId("ns")
	nm.SetName("ns")
	nm.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "ns",
	})
	nss := map[string]*storage.NamespaceMetadata{
		"ns": nm,
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
	protoassert.Equal(t, allowAllIngress, rule)
}

func TestGenerateIngressRule_WithInternetIngress_WithPorts(t *testing.T) {

	internetNode := &node{
		entity: networkgraph.Entity{
			Type: storage.NetworkEntityInfo_INTERNET,
		},
	}

	nm := &storage.NamespaceMetadata{}
	nm.SetId("ns")
	nm.SetName("ns")
	nm.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "ns",
	})
	nss := map[string]*storage.NamespaceMetadata{
		"ns": nm,
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
		storage.NetworkPolicyIngressRule_builder{
			Ports: []*storage.NetworkPolicyPort{
				storage.NetworkPolicyPort_builder{
					Protocol: storage.Protocol_TCP_PROTOCOL,
					Port:     proto.Int32(makeNumPort(443).Port),
				}.Build(),
			},
			From: allowAllIngress.GetFrom(),
		}.Build(),
		storage.NetworkPolicyIngressRule_builder{
			Ports: []*storage.NetworkPolicyPort{
				storage.NetworkPolicyPort_builder{
					Protocol: storage.Protocol_UDP_PROTOCOL,
					Port:     proto.Int32(makeNumPort(53).Port),
				}.Build(),
			},
			From: []*storage.NetworkPolicyPeer{
				storage.NetworkPolicyPeer_builder{
					PodSelector: storage.LabelSelector_builder{
						MatchLabels: map[string]string{
							"app": "foo",
						},
					}.Build(),
				}.Build(),
			},
		}.Build(),
	}

	rules := generateIngressRules(deployment1, nss)
	protoassert.ElementsMatch(t, expectedRules, rules)
}

func TestGenerateIngressRule_WithInternetExposure(t *testing.T) {

	deployment0 := createDeploymentNode("deployment0", "deployment0", "ns", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "deployment1", "ns", nil)

	pc := &storage.PortConfig{}
	pc.SetContainerPort(443)
	pc.SetExposure(storage.PortConfig_EXTERNAL)
	deployment1.deployment.SetPorts([]*storage.PortConfig{
		pc,
	})
	deployment1.incoming[portDesc{}] = &ingressInfo{
		peers: peers{
			deployment0: struct{}{},
		},
	}
	deployment1.populateExposureInfo(false)

	nm := &storage.NamespaceMetadata{}
	nm.SetId("ns")
	nm.SetName("ns")
	nm.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "ns",
	})
	nss := map[string]*storage.NamespaceMetadata{
		"ns": nm,
	}

	rule := generateIngressRule(deployment1, portDesc{}, nss)
	protoassert.Equal(t, allowAllIngress, rule)
}

func TestGenerateIngressRule_WithInternetExposure_WithPorts(t *testing.T) {

	deployment0 := createDeploymentNode("deployment0", "deployment0", "ns", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "deployment1", "ns", nil)

	pc := &storage.PortConfig{}
	pc.SetContainerPort(443)
	pc.SetProtocol("TCP")
	pc.SetExposure(storage.PortConfig_EXTERNAL)
	pc2 := &storage.PortConfig{}
	pc2.SetContainerPort(80)
	pc2.SetProtocol("TCP")
	pc2.SetExposure(storage.PortConfig_NODE)
	deployment1.deployment.SetPorts([]*storage.PortConfig{
		pc,
		pc2,
	})
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

	nm := &storage.NamespaceMetadata{}
	nm.SetId("ns")
	nm.SetName("ns")
	nm.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "ns",
	})
	nss := map[string]*storage.NamespaceMetadata{
		"ns": nm,
	}

	expectedRules := []*storage.NetworkPolicyIngressRule{
		storage.NetworkPolicyIngressRule_builder{
			Ports: []*storage.NetworkPolicyPort{
				storage.NetworkPolicyPort_builder{
					Protocol: storage.Protocol_TCP_PROTOCOL,
					Port:     proto.Int32(makeNumPort(443).Port),
				}.Build(),
			},
			From: allowAllIngress.GetFrom(),
		}.Build(),
		storage.NetworkPolicyIngressRule_builder{
			Ports: []*storage.NetworkPolicyPort{
				storage.NetworkPolicyPort_builder{
					Protocol: storage.Protocol_TCP_PROTOCOL,
					Port:     proto.Int32(makeNumPort(80).Port),
				}.Build(),
			},
			From: allowAllIngress.GetFrom(),
		}.Build(),
		storage.NetworkPolicyIngressRule_builder{
			Ports: []*storage.NetworkPolicyPort{
				storage.NetworkPolicyPort_builder{
					Protocol: storage.Protocol_UDP_PROTOCOL,
					Port:     proto.Int32(makeNumPort(53).Port),
				}.Build(),
			},
			From: []*storage.NetworkPolicyPeer{
				storage.NetworkPolicyPeer_builder{
					PodSelector: storage.LabelSelector_builder{
						MatchLabels: map[string]string{
							"app": "foo",
						},
					}.Build(),
				}.Build(),
			},
		}.Build(),
	}

	rules := generateIngressRules(deployment1, nss)
	protoassert.ElementsMatch(t, expectedRules, rules)
}

func TestGenerateIngressRule_WithoutInternet(t *testing.T) {

	deployment0 := createDeploymentNode("deployment0", "deployment0", "ns1", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "deployment1", "ns2", map[string]string{"app": "bar"})

	tgtDeployment := createDeploymentNode("tgtDeployment", "tgtDeployment", "ns1", nil)
	tgtDeployment.incoming[portDesc{}] = &ingressInfo{
		peers: peers{
			deployment0: struct{}{},
			deployment1: struct{}{},
		},
	}

	nm := &storage.NamespaceMetadata{}
	nm.SetId("ns1")
	nm.SetName("ns1")
	nm.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "ns1",
	})
	nm2 := &storage.NamespaceMetadata{}
	nm2.SetId("ns2")
	nm2.SetName("ns2")
	nm2.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "ns2",
	})
	nss := map[string]*storage.NamespaceMetadata{
		"ns1": nm,
		"ns2": nm2,
	}

	expectedPeers := []*storage.NetworkPolicyPeer{
		storage.NetworkPolicyPeer_builder{
			PodSelector: storage.LabelSelector_builder{
				MatchLabels: map[string]string{"app": "foo"},
			}.Build(),
		}.Build(),
		storage.NetworkPolicyPeer_builder{
			NamespaceSelector: storage.LabelSelector_builder{
				MatchLabels: map[string]string{namespaces.NamespaceNameLabel: "ns2"},
			}.Build(),
			PodSelector: storage.LabelSelector_builder{
				MatchLabels: map[string]string{"app": "bar"},
			}.Build(),
		}.Build(),
	}

	rule := generateIngressRule(tgtDeployment, portDesc{}, nss)
	protoassert.ElementsMatch(t, expectedPeers, rule.GetFrom())
}

func TestGenerateIngressRule_WithoutInternet_WithPorts(t *testing.T) {

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

	nm := &storage.NamespaceMetadata{}
	nm.SetId("ns1")
	nm.SetName("ns1")
	nm.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "ns1",
	})
	nm2 := &storage.NamespaceMetadata{}
	nm2.SetId("ns2")
	nm2.SetName("ns2")
	nm2.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "ns2",
	})
	nss := map[string]*storage.NamespaceMetadata{
		"ns1": nm,
		"ns2": nm2,
	}

	expectedRules := []*storage.NetworkPolicyIngressRule{
		storage.NetworkPolicyIngressRule_builder{
			Ports: []*storage.NetworkPolicyPort{
				storage.NetworkPolicyPort_builder{
					Protocol: storage.Protocol_TCP_PROTOCOL,
					Port:     proto.Int32(makeNumPort(443).Port),
				}.Build(),
			},
			From: []*storage.NetworkPolicyPeer{
				storage.NetworkPolicyPeer_builder{
					PodSelector: storage.LabelSelector_builder{
						MatchLabels: map[string]string{"app": "foo"},
					}.Build(),
				}.Build(),
				storage.NetworkPolicyPeer_builder{
					NamespaceSelector: storage.LabelSelector_builder{
						MatchLabels: map[string]string{namespaces.NamespaceNameLabel: "ns2"},
					}.Build(),
					PodSelector: storage.LabelSelector_builder{
						MatchLabels: map[string]string{"app": "bar"},
					}.Build(),
				}.Build(),
			},
		}.Build(),
		storage.NetworkPolicyIngressRule_builder{
			Ports: []*storage.NetworkPolicyPort{
				storage.NetworkPolicyPort_builder{
					Protocol: storage.Protocol_UDP_PROTOCOL,
					Port:     proto.Int32(makeNumPort(53).Port),
				}.Build(),
			},
			From: []*storage.NetworkPolicyPeer{
				storage.NetworkPolicyPeer_builder{
					PodSelector: storage.LabelSelector_builder{
						MatchLabels: map[string]string{"app": "foo"},
					}.Build(),
				}.Build(),
			},
		}.Build(),
	}

	rules := generateIngressRules(tgtDeployment, nss)
	// Modify rules to make sure that all namespace-local peers appear before all foreign-namespace-peers;
	// otherwise the comparison below might fail.
	for _, rule := range rules {
		sort.SliceStable(rule.GetFrom(), func(i, j int) bool {
			return rule.GetFrom()[i].GetNamespaceSelector() == nil && rule.GetFrom()[j].GetNamespaceSelector() != nil
		})
	}
	protoassert.ElementsMatch(t, expectedRules, rules)
}

func TestGenerateIngressRule_ScopeAlienDeployment(t *testing.T) {

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

	nm := &storage.NamespaceMetadata{}
	nm.SetId("ns1")
	nm.SetName("ns1")
	nm.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "ns1",
	})
	nss := map[string]*storage.NamespaceMetadata{
		"ns": nm,
	}

	npp := &storage.NetworkPolicyPeer{}
	npp.SetNamespaceSelector(&storage.LabelSelector{})
	npp.SetPodSelector(&storage.LabelSelector{})
	expectedPeers := []*storage.NetworkPolicyPeer{
		npp,
	}
	rule := generateIngressRule(tgtDeployment, portDesc{}, nss)
	protoassert.SlicesEqual(t, expectedPeers, rule.GetFrom())
}

func TestGenerateIngressRule_ScopeAlienNSOnly(t *testing.T) {

	deployment0 := createDeploymentNode("deployment0", "deployment0", "ns1", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "deployment1", "ns2", map[string]string{"app": "bar"})
	tgtDeployment := createDeploymentNode("tgtDeployment", "tgtDeployment", "ns1", nil)
	tgtDeployment.incoming[portDesc{}] = &ingressInfo{
		peers: peers{
			deployment0: struct{}{},
			deployment1: struct{}{},
		},
	}

	nm := &storage.NamespaceMetadata{}
	nm.SetId("ns1")
	nm.SetName("ns1")
	nm.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "ns1",
	})
	nss := map[string]*storage.NamespaceMetadata{
		"ns": nm,
	}

	expectedPeers := []*storage.NetworkPolicyPeer{
		storage.NetworkPolicyPeer_builder{
			PodSelector: storage.LabelSelector_builder{
				MatchLabels: map[string]string{"app": "foo"},
			}.Build(),
		}.Build(),
		storage.NetworkPolicyPeer_builder{
			NamespaceSelector: &storage.LabelSelector{},
			PodSelector: storage.LabelSelector_builder{
				MatchLabels: map[string]string{"app": "bar"},
			}.Build(),
		}.Build(),
	}
	rule := generateIngressRule(tgtDeployment, portDesc{}, nss)
	protoassert.ElementsMatch(t, expectedPeers, rule.GetFrom())
}

func TestGenerateIngressRule_FromProtectedNS(t *testing.T) {

	tgtDeployment := createDeploymentNode("tgtDeployment", "tgtDeployment", "ns1", nil)

	deployment0 := createDeploymentNode("deployment0", "deployment0", "kube-system", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "deployment1", "ns2", map[string]string{"app": "bar"})

	tgtDeployment.incoming[portDesc{}] = &ingressInfo{
		peers: peers{
			deployment0: struct{}{},
			deployment1: struct{}{},
		},
	}

	nm := &storage.NamespaceMetadata{}
	nm.SetId("ns1")
	nm.SetName("ns1")
	nm.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "ns1",
	})
	nm2 := &storage.NamespaceMetadata{}
	nm2.SetId("ns2")
	nm2.SetName("ns2")
	nm2.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "ns2",
	})
	nm3 := &storage.NamespaceMetadata{}
	nm3.SetId("kube-system")
	nm3.SetName("kube-system")
	nm3.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "kube-system",
	})
	nss := map[string]*storage.NamespaceMetadata{
		"ns1":         nm,
		"ns2":         nm2,
		"kube-system": nm3,
	}

	expectedPeers := []*storage.NetworkPolicyPeer{
		storage.NetworkPolicyPeer_builder{
			NamespaceSelector: storage.LabelSelector_builder{
				MatchLabels: map[string]string{
					namespaces.NamespaceNameLabel: "kube-system",
				},
			}.Build(),
			PodSelector: storage.LabelSelector_builder{
				MatchLabels: map[string]string{"app": "foo"},
			}.Build(),
		}.Build(),
		storage.NetworkPolicyPeer_builder{
			NamespaceSelector: storage.LabelSelector_builder{
				MatchLabels: map[string]string{
					namespaces.NamespaceNameLabel: "ns2",
				},
			}.Build(),
			PodSelector: storage.LabelSelector_builder{
				MatchLabels: map[string]string{"app": "bar"},
			}.Build(),
		}.Build(),
	}

	rules := generateIngressRules(tgtDeployment, nss)
	require.Len(t, rules, 1)
	protoassert.ElementsMatch(t, expectedPeers, rules[0].GetFrom())
}
