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
		deployments     []*storage.Deployment
		extSrcs         []*storage.NetworkEntityInfo
		peer            *storage.NetworkPolicyPeer
		policyNamespace string
		expectedMatches int
	}{
		{
			name: "zero peer",
			deployments: []*storage.Deployment{
				{
					Namespace:   "default",
					NamespaceId: "default",
				},
			},
			peer:            &storage.NetworkPolicyPeer{},
			policyNamespace: "default",
			expectedMatches: 0,
		},
		{
			name: "match all in same NS peer",
			deployments: []*storage.Deployment{
				{
					Namespace:   "default",
					NamespaceId: "default",
				},
			},
			peer: &storage.NetworkPolicyPeer{
				PodSelector: &storage.LabelSelector{},
			},
			policyNamespace: "default",
			expectedMatches: 1,
		},
		{
			name: "match all in same NS peer - no match",
			deployments: []*storage.Deployment{
				{
					Namespace:   "default",
					NamespaceId: "default",
				},
			},
			peer: &storage.NetworkPolicyPeer{
				PodSelector: &storage.LabelSelector{},
			},
			policyNamespace: "stackrox",
			expectedMatches: 0,
		},
		{
			name: "match all in all NS peer",
			deployments: []*storage.Deployment{
				{
					Namespace:   "default",
					NamespaceId: "default",
				},
			},
			peer: &storage.NetworkPolicyPeer{
				PodSelector:       &storage.LabelSelector{},
				NamespaceSelector: &storage.LabelSelector{},
			},
			policyNamespace: "stackrox",
			expectedMatches: 1,
		},
		{
			name:            "ip block",
			deployments:     []*storage.Deployment{},
			peer:            &storage.NetworkPolicyPeer{IpBlock: &storage.IPBlock{}},
			expectedMatches: 0,
		},
		{
			name: "ip block - matches external source; fully contained",
			deployments: []*storage.Deployment{
				{
					Namespace: "default",
				},
			},
			extSrcs: []*storage.NetworkEntityInfo{
				{
					Desc: &storage.NetworkEntityInfo_ExternalSource_{
						ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
							Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
								Cidr: "192.16.0.0/16",
							},
						},
					},
				},
			},
			peer: &storage.NetworkPolicyPeer{
				IpBlock: &storage.IPBlock{
					Cidr: "192.16.0.0/24",
				},
			},
			expectedMatches: 2,
		},
		{
			name: "ip block - matches external source; partial overlap",
			deployments: []*storage.Deployment{
				{
					Namespace: "default",
				},
			},
			extSrcs: []*storage.NetworkEntityInfo{
				{
					Desc: &storage.NetworkEntityInfo_ExternalSource_{
						ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
							Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
								Cidr: "192.16.0.0/32",
							},
						},
					},
				},
			},
			peer: &storage.NetworkPolicyPeer{
				IpBlock: &storage.IPBlock{
					Cidr: "192.16.0.0/24",
				},
			},
			expectedMatches: 3,
		},
		{
			name: "ip block - does not match external source; matches INTERNET",
			extSrcs: []*storage.NetworkEntityInfo{
				{
					Desc: &storage.NetworkEntityInfo_ExternalSource_{
						ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
							Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
								Cidr: "192.16.0.0/16",
							},
						},
					},
				},
			},
			peer: &storage.NetworkPolicyPeer{
				IpBlock: &storage.IPBlock{
					Cidr: "192.0.0.0/24",
				},
			},
			expectedMatches: 1,
		},
		{
			name: "non match pod selector",
			deployments: []*storage.Deployment{
				{
					PodLabels: map[string]string{
						"key": "value1",
					},
				},
			},
			peer: &storage.NetworkPolicyPeer{
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{
						"key": "value",
					},
				},
			},
			expectedMatches: 0,
		},
		{
			name: "match pod selector",
			deployments: []*storage.Deployment{
				{
					Namespace:   "default",
					NamespaceId: "default",
					PodLabels: map[string]string{
						"key": "value",
					},
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
			expectedMatches: 1,
		},
		{
			name: "match namespace selector",
			deployments: []*storage.Deployment{
				{
					Namespace:   "default",
					NamespaceId: "default",
				},
			},
			peer: &storage.NetworkPolicyPeer{
				NamespaceSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{
						"name": "default",
					},
				},
			},
			policyNamespace: "default",
			expectedMatches: 1,
		},
		{
			name: "non match namespace selector",
			deployments: []*storage.Deployment{
				{
					Namespace:   "default",
					NamespaceId: "default",
				},
			},
			peer: &storage.NetworkPolicyPeer{
				NamespaceSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{
						"key": "value1",
					},
				},
			},
			policyNamespace: "default",
			expectedMatches: 0,
		},
		{
			name: "different namespaces",
			deployments: []*storage.Deployment{
				{
					Namespace:   "default",
					NamespaceId: "default",
				},
			},
			peer: &storage.NetworkPolicyPeer{
				NamespaceSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{
						"key": "value1",
					},
				},
			},
			policyNamespace: "stackrox",
			expectedMatches: 0,
		},
		{
			name: "match namespace and pod selector",
			deployments: []*storage.Deployment{
				{
					Namespace:   "default",
					NamespaceId: "default",
					PodLabels: map[string]string{
						"app": "web",
					},
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
			expectedMatches: 1,
		},
		{
			name: "match namespace but not pod selector",
			deployments: []*storage.Deployment{
				{
					Namespace:   "default",
					NamespaceId: "default",
					PodLabels: map[string]string{
						"app": "backend",
					},
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
			expectedMatches: 0,
		},
		{
			name: "match pod but not namespace selector",
			deployments: []*storage.Deployment{
				{
					Namespace:   "default",
					NamespaceId: "default",
					PodLabels: map[string]string{
						"app": "web",
					},
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
			expectedMatches: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := newGraphBuilder(c.deployments, c.extSrcs, namespacesByID)
			matches := b.evaluatePeer(namespacesByID[c.policyNamespace], c.peer)
			assert.Len(t, matches, c.expectedMatches)
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
			b := newGraphBuilder([]*storage.Deployment{c.d}, nil, namespacesByID)
			b.AddEdgesForNetworkPolicies([]*storage.NetworkPolicy{c.np})
			assert.Equal(t, c.expected, len(b.allDeployments[0].applyingPoliciesIDs) > 0)
		})
	}
}
