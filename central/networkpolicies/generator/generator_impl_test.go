package generator

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	dDSMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	nsDSMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	networkBaselineMocks "github.com/stackrox/rox/central/networkbaseline/datastore/mocks"
	netTreeMgrMocks "github.com/stackrox/rox/central/networkgraph/entity/networktree/mocks"
	nfDSMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	npDSMocks "github.com/stackrox/rox/central/networkpolicies/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	sacTestutils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type generatorTestSuite struct {
	suite.Suite
	generator *generator

	mockCtrl                 *gomock.Controller
	mocksNetworkPolicies     *npDSMocks.MockDataStore
	mockDeployments          *dDSMocks.MockDataStore
	mockNetTreeMgr           *netTreeMgrMocks.MockManager
	mockGlobalFlowDataStore  *nfDSMocks.MockClusterDataStore
	mockNamespaceStore       *nsDSMocks.MockDataStore
	mockNetworkBaselineStore *networkBaselineMocks.MockDataStore
	hasNoneCtx               context.Context
	hasReadCtx               context.Context
	hasWriteCtx              context.Context
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
	{
		Id:        "policy5",
		Name:      "policy5",
		Namespace: "kube-system",
	},
	{
		Id:        "policy6",
		Name:      "policy6",
		Namespace: "kube-system",
		Labels: map[string]string{
			generatedNetworkPolicyLabel: "true",
		},
	},
	{
		Id:        "policy7",
		Name:      "policy7",
		Namespace: "stackrox",
	},
	{
		Id:        "policy8",
		Name:      "policy8",
		Namespace: "stackrox",
		Labels: map[string]string{
			generatedNetworkPolicyLabel: "true",
		},
	},
}

func (s *generatorTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkPolicy, resources.NetworkGraph, resources.Namespace, resources.Deployment)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkPolicy, resources.NetworkGraph)))

	s.mockCtrl = gomock.NewController(s.T())
	s.mocksNetworkPolicies = npDSMocks.NewMockDataStore(s.mockCtrl)
	s.mockDeployments = dDSMocks.NewMockDataStore(s.mockCtrl)
	s.mockNetTreeMgr = netTreeMgrMocks.NewMockManager(s.mockCtrl)
	s.mockGlobalFlowDataStore = nfDSMocks.NewMockClusterDataStore(s.mockCtrl)
	s.mockNamespaceStore = nsDSMocks.NewMockDataStore(s.mockCtrl)
	s.mockNetworkBaselineStore = networkBaselineMocks.NewMockDataStore(s.mockCtrl)

	s.generator = &generator{
		networkPolicies:     s.mocksNetworkPolicies,
		deploymentStore:     s.mockDeployments,
		networkTreeMgr:      s.mockNetTreeMgr,
		globalFlowDataStore: s.mockGlobalFlowDataStore,
		namespacesStore:     s.mockNamespaceStore,
		networkBaselines:    s.mockNetworkBaselineStore,
	}
}

func (s *generatorTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *generatorTestSuite) TestEnforceGetNetworkPolicies_DeleteNone() {
	s.mocksNetworkPolicies.EXPECT().GetNetworkPolicies(s.hasNoneCtx, gomock.Any(), gomock.Any()).Return(nil, nil)

	existing, toDelete, err := s.generator.getNetworkPolicies(s.hasNoneCtx, v1.GenerateNetworkPoliciesRequest_NONE, "cluster")
	s.Nil(existing)
	s.NoError(err)
	s.Empty(toDelete)
}

func (s *generatorTestSuite) TestGetNetworkPolicies_DeleteNone() {
	s.mocksNetworkPolicies.EXPECT().GetNetworkPolicies(s.hasReadCtx, gomock.Any(), gomock.Any()).Return(testNetworkPolicies, nil)

	existing, toDelete, err := s.generator.getNetworkPolicies(s.hasReadCtx, v1.GenerateNetworkPoliciesRequest_NONE, "cluster")
	s.NoError(err)
	s.ElementsMatch(existing, testNetworkPolicies)
	s.Empty(toDelete)
}

func (s *generatorTestSuite) TestGetNetworkPolicies_DeleteGenerated() {
	s.mocksNetworkPolicies.EXPECT().GetNetworkPolicies(s.hasReadCtx, gomock.Any(), gomock.Any()).Return(testNetworkPolicies, nil)

	existing, toDelete, err := s.generator.getNetworkPolicies(s.hasReadCtx, v1.GenerateNetworkPoliciesRequest_GENERATED_ONLY, "cluster")
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
	s.mocksNetworkPolicies.EXPECT().GetNetworkPolicies(s.hasReadCtx, gomock.Any(), gomock.Any()).Return(testNetworkPolicies, nil)

	existing, toDelete, err := s.generator.getNetworkPolicies(s.hasReadCtx, v1.GenerateNetworkPoliciesRequest_ALL, "cluster")
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

	ctxHasDeploymentsAccessMatcher := sacTestutils.ContextWithAccess(sac.ScopeSuffix{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
		sac.ResourceScopeKey(resources.Deployment.Resource),
		sac.ClusterScopeKey("mycluster"),
	})

	s.mockDeployments.EXPECT().SearchRawDeployments(ctxHasDeploymentsAccessMatcher, gomock.Any()).Return(
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

	s.mockNamespaceStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return(
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
	s.mocksNetworkPolicies.EXPECT().GetNetworkPolicies(s.hasReadCtx, clusterIDMatcher, "").Return(
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

	mockFlowStore := nfDSMocks.NewMockFlowDataStore(s.mockCtrl)

	ctxHasNetworkFlowAccessMatcher := sacTestutils.ContextWithAccess(
		sac.ScopeSuffix{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
			sac.ResourceScopeKey(resources.NetworkGraph.Resource),
			sac.ClusterScopeKey("mycluster"),
		})

	mockFlowStore.EXPECT().GetMatchingFlows(ctxHasNetworkFlowAccessMatcher, gomock.Any(), gomock.Eq(ts)).Return(
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
		}, types.TimestampNow(), nil)

	s.mockNetTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), gomock.Any()).Return(nil)
	s.mockNetTreeMgr.EXPECT().GetDefaultNetworkTree(gomock.Any()).Return(nil)
	s.mockGlobalFlowDataStore.EXPECT().GetFlowStore(gomock.Any(), gomock.Eq("mycluster")).Return(mockFlowStore, nil)

	generatedPolicies, toDelete, err := s.generator.Generate(s.hasReadCtx, req)
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

func depFlow(fromID, toID string) *storage.NetworkFlow {
	return &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   toID,
			},
			DstEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   fromID,
			},
		},
	}
}

func (s *generatorTestSuite) TestGenerateWithMaskedUnselectedAndDeleted() {
	// Test setup:
	// Query selects namespace foo and bar (visible), and qux (visible only for deployments, not ns metadata)
	// Third namespace baz is visible but not selected
	// User has no network flow access in namespace bar
	// Namespace foo has deployments:
	// - depA has incoming flows from depB, depD, depE, and deployment depY that was recently deleted
	// - depB has incoming flows from depA and a deployment without access, depX
	// - depC has incoming flows from depA and depF in namespace qux
	// Namespace bar:
	// - depD has incoming flows from depA
	// Namespace baz:
	// - depE has incoming flows from depB
	// Namespace qux:
	// - depF has incoming flows from depC, depD and depG
	// - depG has no flows
	// EXPECT:
	// - netpol for depA allowing depB, depD, depE
	// - netpol for depB allowing all cluster traffic
	// - netpol for depC allowing depA and depF only in pod labels
	// - NO netpol for depD (no netflow access)
	// - NO netpol for depE (not selected)
	// - netpol for qux (don't need NS metadata for netpol generation, only for peers in other namespaces)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.TestScopeCheckerCoreFromFullScopeMap(s.T(),
			sac.TestScopeMap{
				storage.Access_READ_ACCESS: {
					resources.Deployment.Resource: &sac.TestResourceScope{
						Clusters: map[string]*sac.TestClusterScope{
							"mycluster": {Namespaces: []string{"foo", "bar", "baz", "qux"}},
						},
					},
					resources.NetworkGraph.Resource: &sac.TestResourceScope{
						Clusters: map[string]*sac.TestClusterScope{
							"mycluster": {Namespaces: []string{"foo", "baz", "qux"}},
						},
					},
					resources.Namespace.Resource: &sac.TestResourceScope{
						Clusters: map[string]*sac.TestClusterScope{
							"mycluster": {Namespaces: []string{"foo", "bar", "baz"}},
						},
					},
				},
			}))

	ts := types.TimestampNow()
	req := &v1.GenerateNetworkPoliciesRequest{
		ClusterId:        "mycluster",
		Query:            "Namespace: foo,bar,qux",
		DeleteExisting:   v1.GenerateNetworkPoliciesRequest_NONE,
		NetworkDataSince: ts,
	}

	ctxHasAllDeploymentsAccessMatcher := sacTestutils.ContextWithAccess(sac.ScopeSuffix{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
		sac.ResourceScopeKey(resources.Deployment.Resource),
		sac.ClusterScopeKey("mycluster"),
	})

	s.mockDeployments.EXPECT().SearchRawDeployments(gomock.Not(ctxHasAllDeploymentsAccessMatcher), gomock.Any()).Return(
		[]*storage.Deployment{
			{
				Id:        "depA",
				Name:      "depA",
				Namespace: "foo",
				PodLabels: map[string]string{"depID": "A"},
				LabelSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "A"},
				},
			},
			{
				Id:        "depB",
				Name:      "depB",
				Namespace: "foo",
				PodLabels: map[string]string{"depID": "B"},
				LabelSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "B"},
				},
			},
			{
				Id:        "depC",
				Name:      "depC",
				Namespace: "foo",
				PodLabels: map[string]string{"depID": "C"},
				LabelSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "C"},
				},
			},
			{
				Id:        "depD",
				Name:      "depD",
				Namespace: "bar",
				PodLabels: map[string]string{"depID": "D"},
				LabelSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "D"},
				},
			},
			{
				Id:        "depF",
				Name:      "depF",
				Namespace: "qux",
				PodLabels: map[string]string{"depID": "F"},
				LabelSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "F"},
				},
			},
			{
				Id:        "depG",
				Name:      "depG",
				Namespace: "qux",
				PodLabels: map[string]string{"depID": "G"},
				LabelSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "G"},
				},
			},
		}, nil)

	ctxHasAllNamespaceAccessMatcher := sacTestutils.ContextWithAccess(sac.ScopeSuffix{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
		sac.ResourceScopeKey(resources.Namespace.Resource),
		sac.ClusterScopeKey("mycluster"),
	})

	s.mockNamespaceStore.EXPECT().SearchNamespaces(gomock.Not(ctxHasAllNamespaceAccessMatcher), gomock.Any()).Return(
		[]*storage.NamespaceMetadata{
			{
				Id:   "1",
				Name: "foo",
				Labels: map[string]string{
					namespaces.NamespaceNameLabel: "foo",
				},
			},
			{
				Id:   "2",
				Name: "bar",
				Labels: map[string]string{
					namespaces.NamespaceNameLabel: "bar",
				},
			},
			{
				Id:   "3",
				Name: "baz",
				Labels: map[string]string{
					namespaces.NamespaceNameLabel: "baz",
				},
			},
		}, nil,
	)

	// Assume no existing network policies.
	s.mocksNetworkPolicies.EXPECT().GetNetworkPolicies(gomock.Any(), "mycluster", "").Return(nil, nil)

	mockFlowStore := nfDSMocks.NewMockFlowDataStore(s.mockCtrl)

	ctxHasClusterWideNetworkFlowAccessMatcher := sacTestutils.ContextWithAccess(
		sac.ScopeSuffix{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
			sac.ResourceScopeKey(resources.NetworkGraph.Resource),
			sac.ClusterScopeKey("mycluster"),
		})

	mockFlowStore.EXPECT().GetMatchingFlows(ctxHasClusterWideNetworkFlowAccessMatcher, gomock.Any(), gomock.Eq(ts)).Return(
		[]*storage.NetworkFlow{
			depFlow("depA", "depB"),
			depFlow("depA", "depD"),
			depFlow("depA", "depE"),
			depFlow("depA", "depY"),
			depFlow("depB", "depA"),
			depFlow("depB", "depX"),
			depFlow("depC", "depA"),
			depFlow("depC", "depF"),
			depFlow("depD", "depA"),
			// depE flows not relevant
			depFlow("depF", "depC"),
			depFlow("depF", "depD"),
			depFlow("depF", "depG"),
		}, types.TimestampNow(), nil)

	s.mockNetTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), gomock.Any()).Return(nil)
	s.mockNetTreeMgr.EXPECT().GetDefaultNetworkTree(gomock.Any()).Return(nil)
	s.mockGlobalFlowDataStore.EXPECT().GetFlowStore(gomock.Any(), gomock.Eq("mycluster")).Return(mockFlowStore, nil)

	// Expect a query for looking up deployments that were not selected as part of the initial query
	// (visible or invisible).
	s.mockDeployments.EXPECT().GetDeployments(
		gomock.Not(ctxHasAllDeploymentsAccessMatcher),
		// depD is part of the query since it was eliminated as irrelevant before.
		testutils.AssertionMatcher(assert.ElementsMatch, []string{"depD", "depE", "depX", "depY"})).Return(
		[]*storage.Deployment{
			{
				Id:        "depD",
				Name:      "depD",
				Namespace: "bar",
				PodLabels: map[string]string{"depID": "D"},
				LabelSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "D"},
				},
			},
			{
				Id:        "depE",
				Name:      "depE",
				Namespace: "baz",
				PodLabels: map[string]string{"depID": "E"},
				LabelSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "E"},
				},
			},
		}, nil,
	)

	// Expect a query with elevated privileges for looking up deployments that we are still missing info about
	// (either deleted or invisible to the user).
	s.mockDeployments.EXPECT().Search(ctxHasAllDeploymentsAccessMatcher, gomock.Any()).Return(
		[]search.Result{
			{
				ID: "depX",
			},
			// depY was deleted!
		}, nil)

	generatedPolicies, toDelete, err := s.generator.Generate(ctx, req)
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

	// EXPECT:
	// - netpol for depA allowing depB, depD, depE
	// - netpol for depB allowing all cluster traffic
	// - netpol for depC allowing depA and depF only in pod labels
	// - NO netpol for depD (no netflow access)
	// - NO netpol for depE (not selected)
	// - netpol for qux (don't need NS metadata for netpol generation, only for peers in other namespaces)
	expectedPolicies := []*storage.NetworkPolicy{
		// No policy for depA as there already is an existing policy
		{
			Name:      "stackrox-generated-depA",
			Namespace: "foo",
			Spec: &storage.NetworkPolicySpec{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "A"},
				},
				Ingress: []*storage.NetworkPolicyIngressRule{
					{
						From: []*storage.NetworkPolicyPeer{
							{
								PodSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{"depID": "B"},
								},
							},
							{
								NamespaceSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{namespaces.NamespaceNameLabel: "bar"},
								},
								PodSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{"depID": "D"},
								},
							},
							{
								NamespaceSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{namespaces.NamespaceNameLabel: "baz"},
								},
								PodSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{"depID": "E"},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:      "stackrox-generated-depB",
			Namespace: "foo",
			Spec: &storage.NetworkPolicySpec{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "B"},
				},
				Ingress: []*storage.NetworkPolicyIngressRule{
					allowAllPodsAllNS,
				},
			},
		},
		{
			Name:      "stackrox-generated-depC",
			Namespace: "foo",
			Spec: &storage.NetworkPolicySpec{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "C"},
				},
				Ingress: []*storage.NetworkPolicyIngressRule{
					{
						From: []*storage.NetworkPolicyPeer{
							{
								PodSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{
										"depID": "A",
									},
								},
							},
							{
								NamespaceSelector: &storage.LabelSelector{},
								PodSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{
										"depID": "F",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:      "stackrox-generated-depF",
			Namespace: "qux",
			Spec: &storage.NetworkPolicySpec{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "F"},
				},
				Ingress: []*storage.NetworkPolicyIngressRule{
					{
						From: []*storage.NetworkPolicyPeer{
							{
								NamespaceSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{
										namespaces.NamespaceNameLabel: "foo",
									},
								},
								PodSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{
										"depID": "C",
									},
								},
							},
							{
								NamespaceSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{
										namespaces.NamespaceNameLabel: "bar",
									},
								},
								PodSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{
										"depID": "D",
									},
								},
							},
							{
								PodSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{
										"depID": "G",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:      "stackrox-generated-depG",
			Namespace: "qux",
			Spec: &storage.NetworkPolicySpec{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "G"},
				},
				Ingress: nil,
			},
		},
	}

	sortPolicies(expectedPolicies)
	s.Equal(expectedPolicies, generatedPolicies)
}

func (s *generatorTestSuite) TestGenerateFromBaselineForDeployment() {
	skipTest := true
	if skipTest {
		return
	}
	deps := make([]*storage.Deployment, 0, 3)
	for i := 0; i < 3; i++ {
		depID := fmt.Sprintf("deployment%03d", i)
		deps = append(deps, &storage.Deployment{
			Id:          depID,
			Name:        depID,
			Namespace:   "some-namespace",
			ClusterId:   "some-cluster",
			ClusterName: "some-cluster",
			PodLabels: map[string]string{
				"app": depID,
			},
		})
		s.mockDeployments.EXPECT().GetDeployment(gomock.Any(), depID).Return(deps[i], true, nil).AnyTimes()
	}

	s.mockNetworkBaselineStore.EXPECT().GetNetworkBaseline(gomock.Any(), "deployment000").Return(
		&storage.NetworkBaseline{
			DeploymentId: "deployment000",
			ClusterId:    "some-cluster",
			Namespace:    "some-namespace",
			Peers: []*storage.NetworkBaselinePeer{
				{
					Entity: &storage.NetworkEntity{
						Info: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   "deployment001",
						},
					},
					Properties: []*storage.NetworkBaselineConnectionProperties{
						{
							Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
							Port:     80,
							Ingress:  true,
						},
						{
							Protocol: storage.L4Protocol_L4_PROTOCOL_UDP,
							Port:     80,
							Ingress:  true,
						},
					},
				},
				{
					Entity: &storage.NetworkEntity{
						Info: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   "deployment002",
						},
					},
					Properties: []*storage.NetworkBaselineConnectionProperties{
						{
							Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
							Port:     80,
							Ingress:  true,
						},
						{
							Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
							Port:     22,
							Ingress:  false,
						},
					},
				},
			},
		}, true, nil)

	s.mockNamespaceStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return(
		[]*storage.NamespaceMetadata{
			{
				Name: "some-namespace",
			},
		}, nil)

	generated, toDelete, err := s.generator.GenerateFromBaselineForDeployment(s.hasWriteCtx, &v1.GetBaselineGeneratedPolicyForDeploymentRequest{
		DeploymentId:   "deployment000",
		DeleteExisting: v1.GenerateNetworkPoliciesRequest_GENERATED_ONLY,
		IncludePorts:   true,
	})
	s.NoError(err)
	s.Empty(toDelete)

	s.ElementsMatch(generated, []*storage.NetworkPolicy{
		{
			Id:          "",
			Name:        "stackrox-baseline-generated-deployment000",
			ClusterId:   "some-cluster",
			ClusterName: "some-cluster",
			Namespace:   "some-namespace",
			Labels:      map[string]string{"network-policy-generator.stackrox.io/from-baseline": "true"},
			Spec: &storage.NetworkPolicySpec{
				PodSelector: labelSelectorForDeployment(deps[0]),
				Ingress: []*storage.NetworkPolicyIngressRule{
					{
						Ports: []*storage.NetworkPolicyPort{
							{
								PortRef: &storage.NetworkPolicyPort_Port{
									Port: int32(80),
								},
								Protocol: storage.Protocol_TCP_PROTOCOL,
							},
						},
						From: []*storage.NetworkPolicyPeer{
							{
								PodSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{"app": "deployment001"},
								},
							},
							{
								PodSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{"app": "deployment002"},
								},
							},
						},
					},
					{
						Ports: []*storage.NetworkPolicyPort{
							{
								PortRef: &storage.NetworkPolicyPort_Port{
									Port: int32(80),
								},
								Protocol: storage.Protocol_UDP_PROTOCOL,
							},
						},
						From: []*storage.NetworkPolicyPeer{
							{
								PodSelector: &storage.LabelSelector{
									MatchLabels: map[string]string{"app": "deployment001"},
								},
							},
						},
					},
				},
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
			},
			ApiVersion: "networking.k8s.io/v1",
		},
	})
}
