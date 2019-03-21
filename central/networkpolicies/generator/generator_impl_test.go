package generator

import (
	"sort"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	mocks2 "github.com/stackrox/rox/central/deployment/datastore/mocks"
	mocks4 "github.com/stackrox/rox/central/namespace/datastore/mocks"
	mocks3 "github.com/stackrox/rox/central/networkflow/store/mocks"
	"github.com/stackrox/rox/central/networkpolicies/store/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

type generatorTestSuite struct {
	suite.Suite
	generator *generator

	mockCtrl               *gomock.Controller
	mockNetworkPolicyStore *mocks.MockStore
	mockDeploymentsStore   *mocks2.MockDataStore
	mockGlobalFlowStore    *mocks3.MockClusterStore
	mockNamespaceStore     *mocks4.MockDataStore
}

func TestGenerator(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(generatorTestSuite))
}

var testNetworkPolicies = []*storage.NetworkPolicy{
	{
		Id:        "policy1",
		Name:      "policy1",
		Namespace: "ns1",
	},
	{
		Id:        "policy2",
		Name:      "policy2",
		Namespace: "ns1",
		Labels: map[string]string{
			generatedNetworkPolicyLabel: "true",
		},
	},
	{
		Id:        "policy3",
		Name:      "policy3",
		Namespace: "ns2",
	},
	{
		Id:        "policy4",
		Name:      "policy4",
		Namespace: "ns2",
		Labels: map[string]string{
			generatedNetworkPolicyLabel: "true",
		},
	},
}

func (s *generatorTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockNetworkPolicyStore = mocks.NewMockStore(s.mockCtrl)
	s.mockDeploymentsStore = mocks2.NewMockDataStore(s.mockCtrl)
	s.mockGlobalFlowStore = mocks3.NewMockClusterStore(s.mockCtrl)
	s.mockNamespaceStore = mocks4.NewMockDataStore(s.mockCtrl)

	s.generator = &generator{
		networkPolicyStore: s.mockNetworkPolicyStore,
		deploymentStore:    s.mockDeploymentsStore,
		globalFlowStore:    s.mockGlobalFlowStore,
		namespacesStore:    s.mockNamespaceStore,
	}
}

func (s *generatorTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *generatorTestSuite) TestGetNetworkPolicies_DeleteNone() {
	s.mockNetworkPolicyStore.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Any()).Return(testNetworkPolicies, nil)

	existing, toDelete, err := s.generator.getNetworkPolicies(v1.GenerateNetworkPoliciesRequest_NONE, "cluster")
	s.NoError(err)
	s.ElementsMatch(existing, testNetworkPolicies)
	s.Empty(toDelete)
}

func (s *generatorTestSuite) TestGetNetworkPolicies_DeleteGenerated() {
	s.mockNetworkPolicyStore.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Any()).Return(testNetworkPolicies, nil)

	existing, toDelete, err := s.generator.getNetworkPolicies(v1.GenerateNetworkPoliciesRequest_GENERATED_ONLY, "cluster")
	s.NoError(err)
	s.ElementsMatch(existing, []*storage.NetworkPolicy{testNetworkPolicies[0], testNetworkPolicies[2]})
	s.ElementsMatch(toDelete, []*storage.NetworkPolicyReference{
		{
			Namespace: testNetworkPolicies[1].Namespace,
			Name:      testNetworkPolicies[1].Name,
		},
		{
			Namespace: testNetworkPolicies[3].Namespace,
			Name:      testNetworkPolicies[3].Name,
		},
	})
}

func (s *generatorTestSuite) TestGetNetworkPolicies_DeleteAll() {
	s.mockNetworkPolicyStore.EXPECT().GetNetworkPolicies(gomock.Any(), gomock.Any()).Return(testNetworkPolicies, nil)

	existing, toDelete, err := s.generator.getNetworkPolicies(v1.GenerateNetworkPoliciesRequest_ALL, "cluster")
	s.NoError(err)
	s.Empty(existing)
	s.ElementsMatch(toDelete, []*storage.NetworkPolicyReference{
		{
			Namespace: testNetworkPolicies[0].Namespace,
			Name:      testNetworkPolicies[0].Name,
		},
		{
			Namespace: testNetworkPolicies[1].Namespace,
			Name:      testNetworkPolicies[1].Name,
		},
		{
			Namespace: testNetworkPolicies[2].Namespace,
			Name:      testNetworkPolicies[2].Name,
		},
		{
			Namespace: testNetworkPolicies[3].Namespace,
			Name:      testNetworkPolicies[3].Name,
		},
	})
}

func sortPolicies(policies []*storage.NetworkPolicy) {
	for _, policy := range policies {
		for _, ingressRule := range policy.Spec.Ingress {
			sort.Slice(ingressRule.From, func(i, j int) bool {
				return proto.MarshalTextString(ingressRule.From[i]) < proto.MarshalTextString(ingressRule.From[j])
			})
		}
		sort.Slice(policy.Spec.Ingress, func(i, j int) bool {
			return proto.MarshalTextString(policy.Spec.Ingress[i]) < proto.MarshalTextString(policy.Spec.Ingress[j])
		})
	}
	sort.Slice(policies, func(i, j int) bool {
		return proto.MarshalTextString(policies[i]) < proto.MarshalTextString(policies[j])
	})
}

func (s *generatorTestSuite) TestGenerate() {
	ts := types.TimestampNow()
	req := &v1.GenerateNetworkPoliciesRequest{
		ClusterId:        "mycluster",
		DeleteExisting:   v1.GenerateNetworkPoliciesRequest_NONE,
		NetworkDataSince: ts,
	}

	s.mockDeploymentsStore.EXPECT().SearchRawDeployments(gomock.Any()).Return(
		[]*storage.Deployment{
			{
				Id:        "depA",
				Name:      "depA",
				Namespace: "ns1",
				PodLabels: map[string]string{"depID": "A"},
				LabelSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "A"},
				},
			},
			{
				Id:        "depB",
				Name:      "depB",
				Namespace: "ns1",
				PodLabels: map[string]string{"depID": "B"},
				LabelSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "B"},
				},
			},
			{
				Id:        "depC",
				Name:      "depC",
				Namespace: "ns1",
				PodLabels: map[string]string{"depID": "C"},
				LabelSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "C"},
				},
			},
			{
				Id:        "depD",
				Name:      "depD",
				Namespace: "ns2",
				PodLabels: map[string]string{"depID": "D"},
				LabelSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "D"},
				},
			},
		}, nil)

	s.mockNamespaceStore.EXPECT().SearchNamespaces(gomock.Any()).Return(
		[]*storage.NamespaceMetadata{
			{
				Id:   "1",
				Name: "ns1",
				Labels: map[string]string{
					namespaces.NamespaceNameLabel: "ns1",
				},
			},
			{
				Id:   "2",
				Name: "ns2",
				Labels: map[string]string{
					namespaces.NamespaceNameLabel: "ns2",
				},
			},
		}, nil)

	clusterIDMatcher := testutils.PredMatcher("check cluster ID", func(clusterID string) bool { return clusterID == "mycluster" })
	s.mockNetworkPolicyStore.EXPECT().GetNetworkPolicies(clusterIDMatcher, "").Return(
		[]*storage.NetworkPolicy{
			{
				Id:        "np1",
				ClusterId: "mycluster",
				Namespace: "ns1",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{"depID": "A"},
					},
					PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				},
			},
			{
				Id:        "np2",
				ClusterId: "mycluster",
				Namespace: "ns1",
				Spec: &storage.NetworkPolicySpec{
					PodSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{"depID": "B"},
					},
					PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
				},
			},
		}, nil)

	mockFlowStore := mocks3.NewMockFlowStore(s.mockCtrl)
	mockFlowStore.EXPECT().GetAllFlows(gomock.Eq(ts)).Return(
		[]*storage.NetworkFlow{
			{
				Props: &storage.NetworkFlowProperties{
					SrcEntity: &storage.NetworkEntityInfo{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depA",
					},
					DstEntity: &storage.NetworkEntityInfo{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depB",
					},
				},
			},
			{
				Props: &storage.NetworkFlowProperties{
					SrcEntity: &storage.NetworkEntityInfo{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depA",
					},
					DstEntity: &storage.NetworkEntityInfo{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depC",
					},
				},
			},
			{
				Props: &storage.NetworkFlowProperties{
					SrcEntity: &storage.NetworkEntityInfo{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depC",
					},
					DstEntity: &storage.NetworkEntityInfo{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depB",
					},
				},
			},
			{
				Props: &storage.NetworkFlowProperties{
					SrcEntity: &storage.NetworkEntityInfo{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depD",
					},
					DstEntity: &storage.NetworkEntityInfo{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depB",
					},
				},
			},
			{
				Props: &storage.NetworkFlowProperties{
					SrcEntity: &storage.NetworkEntityInfo{
						Type: storage.NetworkEntityInfo_INTERNET,
					},
					DstEntity: &storage.NetworkEntityInfo{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depC",
					},
				},
			},
		}, *types.TimestampNow(), nil)

	s.mockGlobalFlowStore.EXPECT().GetFlowStore(gomock.Eq("mycluster")).Return(mockFlowStore)

	generatedPolicies, toDelete, err := s.generator.Generate(req)
	s.NoError(err)
	s.Empty(toDelete)

	// canonicalize policies, strip out uninteresting fields
	for _, policy := range generatedPolicies {
		s.Equal("true", policy.GetLabels()[generatedNetworkPolicyLabel])
		policy.Labels = nil
		s.Equal(networkPolicyAPIVersion, policy.GetApiVersion())
		policy.ApiVersion = ""
	}

	sortPolicies(generatedPolicies)

	expectedPolicies := []*storage.NetworkPolicy{
		// No policy for depA as there already is an existing policy
		{
			Name:      "stackrox-generated-depB",
			Namespace: "ns1",
			Spec: &storage.NetworkPolicySpec{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "B"},
				},
				Ingress: []*storage.NetworkPolicyIngressRule{
					{
						From: []*storage.NetworkPolicyPeer{
							{
								PodSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{"depID": "A"},
								},
							},
							{
								PodSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{"depID": "C"},
								},
							},
							{
								NamespaceSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{namespaces.NamespaceNameLabel: "ns2"},
								},
								PodSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{"depID": "D"},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:      "stackrox-generated-depC",
			Namespace: "ns1",
			Spec: &storage.NetworkPolicySpec{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "C"},
				},
				Ingress: []*storage.NetworkPolicyIngressRule{
					allowAllIngress,
				},
			},
		},
		{
			Name:      "stackrox-generated-depD",
			Namespace: "ns2",
			Spec: &storage.NetworkPolicySpec{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "D"},
				},
			},
		},
	}

	sortPolicies(expectedPolicies)

	s.Equal(expectedPolicies, generatedPolicies)
}
