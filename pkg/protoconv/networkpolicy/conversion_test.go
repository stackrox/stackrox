package networkpolicy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNetworkPolicyConversion(t *testing.T) {
	// Test empty to empty - this is actually very important as it ensure an empty list is different than a list not specified
	// Kubernetes is very picky about a nil slice vs a non nil slice in terms of the implication in a network policy
	np := &v1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind: "NetworkPolicy",
		},
	}
	protoNetworkPolicy := KubernetesNetworkPolicyWrap{NetworkPolicy: np}.ToRoxNetworkPolicy()
	k8sPolicy := RoxNetworkPolicyWrap{NetworkPolicy: protoNetworkPolicy}.ToKubernetesNetworkPolicy()
	assert.Equal(t, np, k8sPolicy)

	// This is the network policy from the k8s example
	port := intstr.FromInt(5978)
	protocol := coreV1.ProtocolTCP
	np = &v1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "network.k8s.io/v1",
			Kind:       "NetworkPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-network-policy",
			Namespace: "default",
		},
		Spec: v1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"role": "db",
				},
			},
			PolicyTypes: []v1.PolicyType{
				v1.PolicyTypeIngress,
				v1.PolicyTypeEgress,
			},
			Ingress: []v1.NetworkPolicyIngressRule{
				{
					From: []v1.NetworkPolicyPeer{
						{
							IPBlock: &v1.IPBlock{
								CIDR:   "172.17.0.0/16",
								Except: []string{"172.17.1.0/24"},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"project": "myproject",
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"role": "frontend",
								},
							},
						},
					},
					Ports: []v1.NetworkPolicyPort{
						{
							Protocol: &protocol,
							Port:     &port,
						},
					},
				},
			},
			Egress: []v1.NetworkPolicyEgressRule{
				{
					To: []v1.NetworkPolicyPeer{
						{
							IPBlock: &v1.IPBlock{
								CIDR:   "172.17.0.0/16",
								Except: []string{"172.17.1.0/24"},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"project": "myproject",
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"role": "frontend",
								},
							},
						},
					},
					Ports: []v1.NetworkPolicyPort{
						{
							Protocol: &protocol,
							Port:     &port,
						},
					},
				},
			},
		},
	}

	yamlPolicy, err := KubernetesNetworkPolicyWrap{NetworkPolicy: np}.ToYaml()
	assert.NoError(t, err, "yaml generation should succeed")

	protoNetworkPolicy, err = YamlWrap{yaml: yamlPolicy}.ToRoxNetworkPolicy()
	assert.NoError(t, err, "rox policy generation should succeed")

	k8sPolicy, err = YamlWrap{yaml: yamlPolicy}.ToKubernetesNetworkPolicy()
	assert.NoError(t, err, "k8s policy generation should succeed")

	assert.Equal(t, np, k8sPolicy)
}
