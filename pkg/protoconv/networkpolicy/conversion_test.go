package networkpolicy

import (
	"testing"

	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNetworkPolicyConversion(t *testing.T) {
	port := intstr.FromInt(5978)
	protocol := coreV1.ProtocolTCP
	cases := []struct {
		name string
		np   *v1.NetworkPolicy
	}{
		{
			// Test empty to empty - this is actually very important as it ensures an empty list is different from a list not specified
			// Kubernetes is very picky about a nil slice vs a non nil slice in terms of the implication in a network policy
			name: "Empty to Empty",
			np: &v1.NetworkPolicy{
				TypeMeta: metav1.TypeMeta{
					Kind: "NetworkPolicy",
				},
				Spec: v1.NetworkPolicySpec{
					PolicyTypes: []v1.PolicyType{
						v1.PolicyTypeIngress,
						v1.PolicyTypeEgress,
					},
				},
			},
		},
		{
			name: "Empty",
			np: &v1.NetworkPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "networking.k8s.io/v1",
					Kind:       "NetworkPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-np",
				},
				Spec: v1.NetworkPolicySpec{
					PolicyTypes: []v1.PolicyType{
						v1.PolicyTypeIngress,
						v1.PolicyTypeEgress,
					},
				},
			},
		},
		{
			name: "Ingress",
			np: &v1.NetworkPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "networking.k8s.io/v1",
					Kind:       "NetworkPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-np",
				},
				Spec: v1.NetworkPolicySpec{
					PolicyTypes: []v1.PolicyType{
						v1.PolicyTypeIngress,
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
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "environment",
												Operator: metav1.LabelSelectorOpNotIn,
												Values:   []string{"testing", "staging"},
											},
										},
									},
									PodSelector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											"role": "frontend",
										},
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "status",
												Operator: metav1.LabelSelectorOpIn,
												Values:   []string{"active"},
											},
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
			},
		},
		{
			name: "Egress",
			np: &v1.NetworkPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "networking.k8s.io/v1",
					Kind:       "NetworkPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-np",
				},
				Spec: v1.NetworkPolicySpec{
					PolicyTypes: []v1.PolicyType{
						v1.PolicyTypeEgress,
					},
					Egress: []v1.NetworkPolicyEgressRule{
						{
							To: []v1.NetworkPolicyPeer{
								{
									NamespaceSelector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											"project": "myproject",
										},
									},
									IPBlock: &v1.IPBlock{
										CIDR:   "172.17.0.0/16",
										Except: []string{"172.17.1.0/24"},
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
			},
		},
		{
			name: "Ingress and Egress",
			np: &v1.NetworkPolicy{
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
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "status",
								Operator: metav1.LabelSelectorOpNotIn,
								Values:   []string{"disabled", "suspended"},
							},
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
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "environment",
												Operator: metav1.LabelSelectorOpNotIn,
												Values:   []string{"testing", "staging"},
											},
										},
									},
									PodSelector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											"role": "frontend",
										},
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "status",
												Operator: metav1.LabelSelectorOpIn,
												Values:   []string{"active"},
											},
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
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Test k8s conversion is correct
			protoNetworkPolicy := KubernetesNetworkPolicyWrap{NetworkPolicy: c.np}.ToRoxNetworkPolicy()
			k8sPolicy := RoxNetworkPolicyWrap{NetworkPolicy: protoNetworkPolicy}.ToKubernetesNetworkPolicy()
			assert.Equal(t, c.np, k8sPolicy)

			// Test `status` field is not present in the generated yaml
			yaml, err := KubernetesNetworkPolicyWrap{c.np}.ToYaml()
			assert.NoError(t, err)
			uObj, err := k8sutil.UnstructuredFromYAML(yaml)
			assert.NoError(t, err, "yaml generation should succeed")
			_, ok := uObj.Object["status"]
			assert.False(t, ok, "The generate yaml file should not have the 'status' field")
			yaml, err = RoxNetworkPolicyWrap{NetworkPolicy: protoNetworkPolicy}.ToYaml()
			assert.NoError(t, err, "yaml generation should succeed")
			uObj, err = k8sutil.UnstructuredFromYAML(yaml)
			assert.NoError(t, err)
			_, ok = uObj.Object["status"]
			assert.False(t, ok, "The generate yaml file should not have the 'status' field")

			// Generation from yaml k8sNetworkPolicy
			k8sPolicies, err := YamlWrap{Yaml: yaml}.ToKubernetesNetworkPolicies()
			assert.NoError(t, err, "k8s policy generation should succeed")
			assert.Equal(t, 1, len(k8sPolicies), "expected one policy from the yaml")
			assert.Equal(t, c.np, k8sPolicies[0])

			// Generation from yaml to roxNetworkPolicy
			protoNetworkPolicies, err := YamlWrap{Yaml: yaml}.ToRoxNetworkPolicies()
			assert.NoError(t, err, "k8s policy generation should succeed")
			assert.Equal(t, 1, len(protoNetworkPolicies), "expected one policy from the yaml")

			// Check that yaml->rox, and k8s->rox creates the same network policy
			protoNetworkPolicyFromK8s := KubernetesNetworkPolicyWrap{c.np}.ToRoxNetworkPolicy()
			assert.Equal(t, protoNetworkPolicyFromK8s, protoNetworkPolicies[0])
		})
	}
}
