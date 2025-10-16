package graph

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestMatchPolicyPeer(t *testing.T) {

	t1, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{
		storage.NetworkEntityInfo_builder{
			Id:   "1",
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
				Cidr: proto.String("192.168.0.0/16"),
			}.Build(),
		}.Build(),
	})
	assert.NoError(t, err)
	t2, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{
		storage.NetworkEntityInfo_builder{
			Id:   "2",
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
				Cidr: proto.String("192.168.0.0/32"),
			}.Build(),
		}.Build(),
	})
	assert.NoError(t, err)
	t3, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{
		storage.NetworkEntityInfo_builder{
			Id:   "1",
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
				Cidr: proto.String("192.168.10.0/24"),
			}.Build(),
		}.Build(),
		storage.NetworkEntityInfo_builder{
			Id:   "2",
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
				Cidr: proto.String("192.168.15.0/24"),
			}.Build(),
		}.Build(),
	})
	assert.NoError(t, err)
	t4, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{
		storage.NetworkEntityInfo_builder{
			Id:   "3",
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
				Cidr: proto.String("30.30.0.0/32"),
			}.Build(),
		}.Build(),
	})
	assert.NoError(t, err)

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
				storage.Deployment_builder{
					Namespace:   "default",
					NamespaceId: "default",
				}.Build(),
			},
			peer:            &storage.NetworkPolicyPeer{},
			policyNamespace: "default",
			expectedMatches: nil,
		},
		{
			name: "match all in same NS peer",
			deployments: []*storage.Deployment{
				storage.Deployment_builder{
					Id:          "DEP1",
					Namespace:   "default",
					NamespaceId: "default",
				}.Build(),
			},
			peer: storage.NetworkPolicyPeer_builder{
				PodSelector: &storage.LabelSelector{},
			}.Build(),
			policyNamespace: "default",
			expectedMatches: []expectedMatch{{id: "DEP1", matchType: storage.NetworkEntityInfo_DEPLOYMENT}},
		},
		{
			name: "match all in same NS peer - no match",
			deployments: []*storage.Deployment{
				storage.Deployment_builder{
					Namespace:   "default",
					NamespaceId: "default",
				}.Build(),
			},
			peer: storage.NetworkPolicyPeer_builder{
				PodSelector: &storage.LabelSelector{},
			}.Build(),
			policyNamespace: "stackrox",
			expectedMatches: nil,
		},
		{
			name: "match all in all NS peer",
			deployments: []*storage.Deployment{
				storage.Deployment_builder{
					Id:          "DEP1",
					Namespace:   "default",
					NamespaceId: "default",
				}.Build(),
			},
			peer: storage.NetworkPolicyPeer_builder{
				PodSelector:       &storage.LabelSelector{},
				NamespaceSelector: &storage.LabelSelector{},
			}.Build(),
			policyNamespace: "stackrox",
			expectedMatches: []expectedMatch{{id: "DEP1", matchType: storage.NetworkEntityInfo_DEPLOYMENT}},
		},
		{
			name:            "ip block",
			deployments:     []*storage.Deployment{},
			peer:            storage.NetworkPolicyPeer_builder{IpBlock: &storage.IPBlock{}}.Build(),
			expectedMatches: nil,
		},
		{
			name: "ip block - external source fully contains ip block; match deployments and external sources",
			deployments: []*storage.Deployment{
				storage.Deployment_builder{
					Id:        "DEP1",
					Namespace: "default",
				}.Build(),
			},
			networkTree: t1,
			peer: storage.NetworkPolicyPeer_builder{
				IpBlock: storage.IPBlock_builder{
					Cidr: "192.168.0.0/24",
				}.Build(),
			}.Build(),
			expectedMatches: []expectedMatch{
				{id: "DEP1", matchType: storage.NetworkEntityInfo_DEPLOYMENT},
				{id: "1", matchType: storage.NetworkEntityInfo_EXTERNAL_SOURCE},
			},
		},
		{
			name:        "ip block - external source fully contains ip block; match only external source",
			networkTree: t1,
			peer: storage.NetworkPolicyPeer_builder{
				IpBlock: storage.IPBlock_builder{
					Cidr: "192.168.0.0/24",
				}.Build(),
			}.Build(),
			expectedMatches: []expectedMatch{
				{id: "1", matchType: storage.NetworkEntityInfo_EXTERNAL_SOURCE},
			},
		},
		{
			name: "ip block - ip block fully contains external source; match deployments, external sources and INTERNET",
			deployments: []*storage.Deployment{
				storage.Deployment_builder{
					Id:        "DEP1",
					Namespace: "default",
				}.Build(),
			},
			networkTree: t2,
			peer: storage.NetworkPolicyPeer_builder{
				IpBlock: storage.IPBlock_builder{
					Cidr: "192.168.0.0/24",
				}.Build(),
			}.Build(),
			expectedMatches: []expectedMatch{
				{id: "DEP1", matchType: storage.NetworkEntityInfo_DEPLOYMENT},
				{id: "2", matchType: storage.NetworkEntityInfo_EXTERNAL_SOURCE},
				{id: networkgraph.InternetExternalSourceID, matchType: storage.NetworkEntityInfo_INTERNET},
			},
		},
		{
			name:        "ip block - ip block fully contains external source; match INTERNET and exclude except networks",
			networkTree: t3,
			peer: storage.NetworkPolicyPeer_builder{
				IpBlock: storage.IPBlock_builder{
					Cidr:   "192.168.0.0/16",
					Except: []string{"192.168.15.0/22"},
				}.Build(),
			}.Build(),
			expectedMatches: []expectedMatch{
				{id: "1", matchType: storage.NetworkEntityInfo_EXTERNAL_SOURCE},
				{id: networkgraph.InternetExternalSourceID, matchType: storage.NetworkEntityInfo_INTERNET},
			},
		},
		{
			name:        "ip block - does not match external source; matches INTERNET",
			networkTree: t1,
			peer: storage.NetworkPolicyPeer_builder{
				IpBlock: storage.IPBlock_builder{
					Cidr: "192.0.0.0/24",
				}.Build(),
			}.Build(),
			expectedMatches: []expectedMatch{
				{id: networkgraph.InternetExternalSourceID, matchType: storage.NetworkEntityInfo_INTERNET},
			},
		},
		{
			name:        "ip block - matches public IP CIDR block and excludes cluster deployments",
			networkTree: t4,
			deployments: []*storage.Deployment{
				storage.Deployment_builder{
					Id:        "DEP1",
					Namespace: "default",
				}.Build(),
			},
			peer: storage.NetworkPolicyPeer_builder{
				IpBlock: storage.IPBlock_builder{
					Cidr: "30.30.0.0/24",
				}.Build(),
			}.Build(),
			expectedMatches: []expectedMatch{
				{id: "3", matchType: storage.NetworkEntityInfo_EXTERNAL_SOURCE},
				{id: networkgraph.InternetExternalSourceID, matchType: storage.NetworkEntityInfo_INTERNET},
			},
		},
		{
			name: "non match pod selector",
			deployments: []*storage.Deployment{
				storage.Deployment_builder{
					PodLabels: map[string]string{
						"key": "value1",
					},
				}.Build(),
			},
			peer: storage.NetworkPolicyPeer_builder{
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{
						"key": "value",
					},
				}.Build(),
			}.Build(),
			expectedMatches: nil,
		},
		{
			name: "match pod selector",
			deployments: []*storage.Deployment{
				storage.Deployment_builder{
					Id:          "DEP1",
					Namespace:   "default",
					NamespaceId: "default",
					PodLabels: map[string]string{
						"key": "value",
					},
				}.Build(),
			},
			peer: storage.NetworkPolicyPeer_builder{
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{
						"key": "value",
					},
				}.Build(),
			}.Build(),
			policyNamespace: "default",
			expectedMatches: []expectedMatch{{id: "DEP1", matchType: storage.NetworkEntityInfo_DEPLOYMENT}},
		},
		{
			name: "match namespace selector",
			deployments: []*storage.Deployment{
				storage.Deployment_builder{
					Id:          "DEP1",
					Namespace:   "default",
					NamespaceId: "default",
				}.Build(),
			},
			peer: storage.NetworkPolicyPeer_builder{
				NamespaceSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{
						"name": "default",
					},
				}.Build(),
			}.Build(),
			policyNamespace: "default",
			expectedMatches: []expectedMatch{{id: "DEP1", matchType: storage.NetworkEntityInfo_DEPLOYMENT}},
		},
		{
			name: "non match namespace selector",
			deployments: []*storage.Deployment{
				storage.Deployment_builder{
					Namespace:   "default",
					NamespaceId: "default",
				}.Build(),
			},
			peer: storage.NetworkPolicyPeer_builder{
				NamespaceSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{
						"key": "value1",
					},
				}.Build(),
			}.Build(),
			policyNamespace: "default",
			expectedMatches: nil,
		},
		{
			name: "different namespaces",
			deployments: []*storage.Deployment{
				storage.Deployment_builder{
					Namespace:   "default",
					NamespaceId: "default",
				}.Build(),
			},
			peer: storage.NetworkPolicyPeer_builder{
				NamespaceSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{
						"key": "value1",
					},
				}.Build(),
			}.Build(),
			policyNamespace: "stackrox",
			expectedMatches: nil,
		},
		{
			name: "match namespace and pod selector",
			deployments: []*storage.Deployment{
				storage.Deployment_builder{
					Id:          "DEP1",
					Namespace:   "default",
					NamespaceId: "default",
					PodLabels: map[string]string{
						"app": "web",
					},
				}.Build(),
			},
			peer: storage.NetworkPolicyPeer_builder{
				NamespaceSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{
						"name": "default",
					},
				}.Build(),
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{
						"app": "web",
					},
				}.Build(),
			}.Build(),
			policyNamespace: "default",
			expectedMatches: []expectedMatch{{id: "DEP1", matchType: storage.NetworkEntityInfo_DEPLOYMENT}},
		},
		{
			name: "match namespace but not pod selector",
			deployments: []*storage.Deployment{
				storage.Deployment_builder{
					Namespace:   "default",
					NamespaceId: "default",
					PodLabels: map[string]string{
						"app": "backend",
					},
				}.Build(),
			},
			peer: storage.NetworkPolicyPeer_builder{
				NamespaceSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{
						"name": "default",
					},
				}.Build(),
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{
						"app": "web",
					},
				}.Build(),
			}.Build(),
			policyNamespace: "default",
			expectedMatches: nil,
		},
		{
			name: "match pod but not namespace selector",
			deployments: []*storage.Deployment{
				storage.Deployment_builder{
					Namespace:   "default",
					NamespaceId: "default",
					PodLabels: map[string]string{
						"app": "web",
					},
				}.Build(),
			},
			peer: storage.NetworkPolicyPeer_builder{
				NamespaceSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{
						"name": "stackrox",
					},
				}.Build(),
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{
						"app": "web",
					},
				}.Build(),
			}.Build(),
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
						formattedMatch.id = match.deployment.GetId()
					} else if match.extSrc != nil {
						formattedMatch.id = match.extSrc.GetId()
						if match.extSrc.GetId() == networkgraph.InternetExternalSourceID {
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

	cases := []struct {
		name     string
		d        *storage.Deployment
		np       *storage.NetworkPolicy
		expected bool
	}{
		{
			name: "namespace doesn't match source",
			d: storage.Deployment_builder{
				Namespace:   "default",
				NamespaceId: "default",
			}.Build(),
			np: storage.NetworkPolicy_builder{
				Namespace: "stackrox",
			}.Build(),
			expected: false,
		},
		{
			name: "pod selector doesn't match",
			d: storage.Deployment_builder{
				PodLabels: map[string]string{
					"key1": "value1",
				},
				Namespace:   "default",
				NamespaceId: "default",
			}.Build(),
			np: storage.NetworkPolicy_builder{
				Namespace: "default",
				Spec: storage.NetworkPolicySpec_builder{
					PodSelector: storage.LabelSelector_builder{
						MatchLabels: map[string]string{
							"key1": "value2",
						},
					}.Build(),
				}.Build(),
			}.Build(),
			expected: false,
		},
		{
			name: "all matches - has ingress",
			d: storage.Deployment_builder{
				PodLabels: map[string]string{
					"key1": "value1",
				},
				Namespace:   "default",
				NamespaceId: "default",
			}.Build(),
			np: storage.NetworkPolicy_builder{
				Namespace: "default",
				Spec: storage.NetworkPolicySpec_builder{
					PodSelector: storage.LabelSelector_builder{
						MatchLabels: map[string]string{
							"key1": "value1",
						},
					}.Build(),
				}.Build(),
			}.Build(),
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
