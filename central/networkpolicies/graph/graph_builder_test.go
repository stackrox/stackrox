package graph

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestMatchPolicyPeer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		deployment      *storage.Deployment
		peer            *storage.NetworkPolicyPeer
		policyNamespace string
		expected        bool
	}{
		{
			name: "zero peer",
			deployment: &storage.Deployment{
				Namespace:   "default",
				NamespaceId: "default",
			},
			peer:            &storage.NetworkPolicyPeer{},
			policyNamespace: "default",
			expected:        false,
		},
		{
			name: "match all in same NS peer",
			deployment: &storage.Deployment{
				Namespace:   "default",
				NamespaceId: "default",
			},
			peer: &storage.NetworkPolicyPeer{
				PodSelector: &storage.LabelSelector{},
			},
			policyNamespace: "default",
			expected:        true,
		},
		{
			name: "match all in same NS peer - no match",
			deployment: &storage.Deployment{
				Namespace:   "default",
				NamespaceId: "default",
			},
			peer: &storage.NetworkPolicyPeer{
				PodSelector: &storage.LabelSelector{},
			},
			policyNamespace: "stackrox",
			expected:        false,
		},
		{
			name: "match all in all NS peer",
			deployment: &storage.Deployment{
				Namespace:   "default",
				NamespaceId: "default",
			},
			peer: &storage.NetworkPolicyPeer{
				PodSelector:       &storage.LabelSelector{},
				NamespaceSelector: &storage.LabelSelector{},
			},
			policyNamespace: "stackrox",
			expected:        true,
		},
		{
			name:       "ip block",
			deployment: &storage.Deployment{},
			peer:       &storage.NetworkPolicyPeer{IpBlock: &storage.IPBlock{}},
			expected:   true,
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
				Namespace:   "default",
				NamespaceId: "default",
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
			policyNamespace: "default",
			expected:        true,
		},
		{
			name: "match namespace selector",
			deployment: &storage.Deployment{
				Namespace:   "default",
				NamespaceId: "default",
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
				Namespace:   "default",
				NamespaceId: "default",
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
				Namespace:   "default",
				NamespaceId: "default",
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
		{
			name: "match namespace and pod selector",
			deployment: &storage.Deployment{
				Namespace:   "default",
				NamespaceId: "default",
				PodLabels: map[string]string{
					"app": "web",
				},
			},
			peer: &storage.NetworkPolicyPeer{
				NamespaceSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{
						"name": "default",
					},
				},
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{
						"app": "web",
					},
				},
			},
			policyNamespace: "default",
			expected:        true,
		},
		{
			name: "match namespace but not pod selector",
			deployment: &storage.Deployment{
				Namespace:   "default",
				NamespaceId: "default",
				PodLabels: map[string]string{
					"app": "backend",
				},
			},
			peer: &storage.NetworkPolicyPeer{
				NamespaceSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{
						"name": "default",
					},
				},
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{
						"app": "web",
					},
				},
			},
			policyNamespace: "default",
			expected:        false,
		},
		{
			name: "match pod but not namespace selector",
			deployment: &storage.Deployment{
				Namespace:   "default",
				NamespaceId: "default",
				PodLabels: map[string]string{
					"app": "web",
				},
			},
			peer: &storage.NetworkPolicyPeer{
				NamespaceSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{
						"name": "stackrox",
					},
				},
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{
						"app": "web",
					},
				},
			},
			policyNamespace: "default",
			expected:        false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := newGraphBuilder([]*storage.Deployment{c.deployment}, namespacesByID)
			matches := b.evaluatePeer(namespacesByID[c.policyNamespace], c.peer)
			assert.Equal(t, c.expected, len(matches) > 0)
		})
	}
}

func TestIngressNetworkPolicySelectorAppliesToDeployment(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		d        *storage.Deployment
		np       *storage.NetworkPolicy
		expected bool
	}{
		{
			name: "namespace doesn't match source",
			d: &storage.Deployment{
				Namespace:   "default",
				NamespaceId: "default",
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
				Namespace:   "default",
				NamespaceId: "default",
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
				Namespace:   "default",
				NamespaceId: "default",
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
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := newGraphBuilder([]*storage.Deployment{c.d}, namespacesByID)
			b.AddEdgesForNetworkPolicies([]*storage.NetworkPolicy{c.np})
			assert.Equal(t, c.expected, len(b.allDeployments[0].applyingPoliciesIDs) > 0)
		})
	}
}
