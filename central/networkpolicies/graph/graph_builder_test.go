package graph

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stretchr/testify/assert"
)

func TestMatchPolicyPeer(t *testing.T) {
	t.Parallel()

	t1, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{
		{
			Id:   "1",
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "192.168.0.0/16",
					},
				},
			},
		},
	})
	assert.NoError(t, err)
	t2, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{
		{
			Id:   "2",
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "192.168.0.0/32",
					},
				},
			},
		},
	})
	assert.NoError(t, err)
	t3, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{
		{
			Id:   "1",
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "192.168.10.0/24",
					},
				},
			},
		},
		{
			Id:   "2",
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "192.168.15.0/24",
					},
				},
			},
		},
	})
	assert.NoError(t, err)
	t4, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{
		{
			Id:   "3",
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "30.30.0.0/32",
					},
				},
			},
		},
	})

	_, _, _, _ = t1, t2, t3, t4

	type expectedMatch struct {
		id        string
		matchType storage.NetworkEntityInfo_Type
	}

	cases := []struct {
		name            string
		deployments     []*storage.Deployment
		networkTree     tree.ReadOnlyNetworkTree
		peer            *storage.NetworkPolicyPeer
		policyNamespace string
		expectedMatches []expectedMatch
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
			expectedMatches: nil,
		},
		{
			name: "match all in same NS peer",
			deployments: []*storage.Deployment{
				{
					Id:          "DEP1",
					Namespace:   "default",
					NamespaceId: "default",
				},
			},
			peer: &storage.NetworkPolicyPeer{
				PodSelector: &storage.LabelSelector{},
			},
			policyNamespace: "default",
			expectedMatches: []expectedMatch{{id: "DEP1", matchType: storage.NetworkEntityInfo_DEPLOYMENT}},
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
			expectedMatches: nil,
		},
		{
			name: "match all in all NS peer",
			deployments: []*storage.Deployment{
				{
					Id:          "DEP1",
					Namespace:   "default",
					NamespaceId: "default",
				},
			},
			peer: &storage.NetworkPolicyPeer{
				PodSelector:       &storage.LabelSelector{},
				NamespaceSelector: &storage.LabelSelector{},
			},
			policyNamespace: "stackrox",
			expectedMatches: []expectedMatch{{id: "DEP1", matchType: storage.NetworkEntityInfo_DEPLOYMENT}},
		},
		{
			name:            "ip block",
			deployments:     []*storage.Deployment{},
			peer:            &storage.NetworkPolicyPeer{IpBlock: &storage.IPBlock{}},
			expectedMatches: nil,
		},
		{
			name: "ip block - external source fully contains ip block; match deployments and external sources",
			deployments: []*storage.Deployment{
				{
					Id:        "DEP1",
					Namespace: "default",
				},
			},
			networkTree: t1,
			peer: &storage.NetworkPolicyPeer{
				IpBlock: &storage.IPBlock{
					Cidr: "192.168.0.0/24",
				},
			},
			expectedMatches: []expectedMatch{
				{id: "DEP1", matchType: storage.NetworkEntityInfo_DEPLOYMENT},
				{id: "1", matchType: storage.NetworkEntityInfo_EXTERNAL_SOURCE},
			},
		},
		{
			name:        "ip block - external source fully contains ip block; match only external source",
			networkTree: t1,
			peer: &storage.NetworkPolicyPeer{
				IpBlock: &storage.IPBlock{
					Cidr: "192.168.0.0/24",
				},
			},
			expectedMatches: []expectedMatch{
				{id: "1", matchType: storage.NetworkEntityInfo_EXTERNAL_SOURCE},
			},
		},
		{
			name: "ip block - ip block fully contains external source; match deployments, external sources and INTERNET",
			deployments: []*storage.Deployment{
				{
					Id:        "DEP1",
					Namespace: "default",
				},
			},
			networkTree: t2,
			peer: &storage.NetworkPolicyPeer{
				IpBlock: &storage.IPBlock{
					Cidr: "192.168.0.0/24",
				},
			},
			expectedMatches: []expectedMatch{
				{id: "DEP1", matchType: storage.NetworkEntityInfo_DEPLOYMENT},
				{id: "2", matchType: storage.NetworkEntityInfo_EXTERNAL_SOURCE},
				{id: networkgraph.InternetExternalSourceID, matchType: storage.NetworkEntityInfo_INTERNET},
			},
		},
		{
			name:        "ip block - ip block fully contains external source; match INTERNET and exclude except networks",
			networkTree: t3,
			peer: &storage.NetworkPolicyPeer{
				IpBlock: &storage.IPBlock{
					Cidr:   "192.168.0.0/16",
					Except: []string{"192.168.15.0/22"},
				},
			},
			expectedMatches: []expectedMatch{
				{id: "1", matchType: storage.NetworkEntityInfo_EXTERNAL_SOURCE},
				{id: networkgraph.InternetExternalSourceID, matchType: storage.NetworkEntityInfo_INTERNET},
			},
		},
		{
			name:        "ip block - does not match external source; matches INTERNET",
			networkTree: t1,
			peer: &storage.NetworkPolicyPeer{
				IpBlock: &storage.IPBlock{
					Cidr: "192.0.0.0/24",
				},
			},
			expectedMatches: []expectedMatch{
				{id: networkgraph.InternetExternalSourceID, matchType: storage.NetworkEntityInfo_INTERNET},
			},
		},
		{
			name:        "ip block - matches public IP CIDR block and excludes cluster deployments",
			networkTree: t4,
			deployments: []*storage.Deployment{
				{
					Id:        "DEP1",
					Namespace: "default",
				},
			},
			peer: &storage.NetworkPolicyPeer{
				IpBlock: &storage.IPBlock{
					Cidr: "30.30.0.0/24",
				},
			},
			expectedMatches: []expectedMatch{
				{id: "3", matchType: storage.NetworkEntityInfo_EXTERNAL_SOURCE},
				{id: networkgraph.InternetExternalSourceID, matchType: storage.NetworkEntityInfo_INTERNET},
			},
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
			expectedMatches: nil,
		},
		{
			name: "match pod selector",
			deployments: []*storage.Deployment{
				{
					Id:          "DEP1",
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
			expectedMatches: []expectedMatch{{id: "DEP1", matchType: storage.NetworkEntityInfo_DEPLOYMENT}},
		},
		{
			name: "match namespace selector",
			deployments: []*storage.Deployment{
				{
					Id:          "DEP1",
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
			expectedMatches: []expectedMatch{{id: "DEP1", matchType: storage.NetworkEntityInfo_DEPLOYMENT}},
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
			expectedMatches: nil,
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
			expectedMatches: nil,
		},
		{
			name: "match namespace and pod selector",
			deployments: []*storage.Deployment{
				{
					Id:          "DEP1",
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
			expectedMatches: []expectedMatch{{id: "DEP1", matchType: storage.NetworkEntityInfo_DEPLOYMENT}},
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
			expectedMatches: nil,
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
			expectedMatches: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := newGraphBuilder(nil, c.deployments, c.networkTree, namespacesByID)
			// Run the test multiple times to make sure that the graph builder is not put into a bad state the first time,
			// and is capable of returning consistent results.
			for i := 0; i < 2; i++ {
				matches := b.evaluatePeer(namespacesByID[c.policyNamespace], c.peer)
				formattedMatches := make([]expectedMatch, 0, len(matches))
				for _, match := range matches {
					var formattedMatch expectedMatch
					if match.deployment != nil {
						formattedMatch.matchType = storage.NetworkEntityInfo_DEPLOYMENT
						formattedMatch.id = match.deployment.Id
					} else if match.extSrc != nil {
						formattedMatch.id = match.extSrc.Id
						if match.extSrc.Id == networkgraph.InternetExternalSourceID {
							formattedMatch.matchType = storage.NetworkEntityInfo_INTERNET
						} else {
							formattedMatch.matchType = storage.NetworkEntityInfo_EXTERNAL_SOURCE
						}
					}
					formattedMatches = append(formattedMatches, formattedMatch)
				}
				assert.ElementsMatch(t, formattedMatches, c.expectedMatches)
			}
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
			b := newGraphBuilder(nil, []*storage.Deployment{c.d}, nil, namespacesByID)
			b.AddEdgesForNetworkPolicies([]*storage.NetworkPolicy{c.np})
			assert.Equal(t, c.expected, len(b.allDeployments[0].applyingPoliciesIDs) > 0)
		})
	}
}
