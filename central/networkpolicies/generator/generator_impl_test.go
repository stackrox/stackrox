package generator

import (
	"context"
	"sort"
	"testing"
	"time"

	dDSMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	nsDSMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	networkBaselineMocks "github.com/stackrox/rox/central/networkbaseline/datastore/mocks"
	netTreeMgrMocks "github.com/stackrox/rox/central/networkgraph/entity/networktree/mocks"
	nfDSMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	npDSMocks "github.com/stackrox/rox/central/networkpolicies/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
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
	suite.Run(t, new(generatorTestSuite))
}

var testNetworkPolicies = []*storage.NetworkPolicy{
	storage.NetworkPolicy_builder{
		Id:        "policy1",
		Name:      "policy1",
		Namespace: "ns1",
	}.Build(),
	storage.NetworkPolicy_builder{
		Id:        "policy2",
		Name:      "policy2",
		Namespace: "ns1",
		Labels: map[string]string{
			generatedNetworkPolicyLabel: "true",
		},
	}.Build(),
	storage.NetworkPolicy_builder{
		Id:        "policy3",
		Name:      "policy3",
		Namespace: "ns2",
	}.Build(),
	storage.NetworkPolicy_builder{
		Id:        "policy4",
		Name:      "policy4",
		Namespace: "ns2",
		Labels: map[string]string{
			generatedNetworkPolicyLabel: "true",
		},
	}.Build(),
	storage.NetworkPolicy_builder{
		Id:        "policy5",
		Name:      "policy5",
		Namespace: "kube-system",
	}.Build(),
	storage.NetworkPolicy_builder{
		Id:        "policy6",
		Name:      "policy6",
		Namespace: "kube-system",
		Labels: map[string]string{
			generatedNetworkPolicyLabel: "true",
		},
	}.Build(),
	storage.NetworkPolicy_builder{
		Id:        "policy7",
		Name:      "policy7",
		Namespace: "stackrox",
	}.Build(),
	storage.NetworkPolicy_builder{
		Id:        "policy8",
		Name:      "policy8",
		Namespace: "stackrox",
		Labels: map[string]string{
			generatedNetworkPolicyLabel: "true",
		},
	}.Build(),
}

func (s *generatorTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkPolicy, resources.NetworkGraph, resources.Namespace, resources.Deployment)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
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
	protoassert.ElementsMatch(s.T(), existing, testNetworkPolicies)
	s.Empty(toDelete)
}

func (s *generatorTestSuite) TestGetNetworkPolicies_DeleteGenerated() {
	s.mocksNetworkPolicies.EXPECT().GetNetworkPolicies(s.hasReadCtx, gomock.Any(), gomock.Any()).Return(testNetworkPolicies, nil)

	existing, toDelete, err := s.generator.getNetworkPolicies(s.hasReadCtx, v1.GenerateNetworkPoliciesRequest_GENERATED_ONLY, "cluster")
	s.NoError(err)
	protoassert.ElementsMatch(s.T(), existing, []*storage.NetworkPolicy{testNetworkPolicies[0], testNetworkPolicies[2]})
	npr := &storage.NetworkPolicyReference{}
	npr.SetNamespace(testNetworkPolicies[1].GetNamespace())
	npr.SetName(testNetworkPolicies[1].GetName())
	npr2 := &storage.NetworkPolicyReference{}
	npr2.SetNamespace(testNetworkPolicies[3].GetNamespace())
	npr2.SetName(testNetworkPolicies[3].GetName())
	protoassert.ElementsMatch(s.T(), toDelete, []*storage.NetworkPolicyReference{
		npr,
		npr2,
	})
}

func (s *generatorTestSuite) TestGetNetworkPolicies_DeleteAll() {
	s.mocksNetworkPolicies.EXPECT().GetNetworkPolicies(s.hasReadCtx, gomock.Any(), gomock.Any()).Return(testNetworkPolicies, nil)

	existing, toDelete, err := s.generator.getNetworkPolicies(s.hasReadCtx, v1.GenerateNetworkPoliciesRequest_ALL, "cluster")
	s.NoError(err)
	s.Empty(existing)
	protoassert.ElementsMatch(s.T(), toDelete, []*storage.NetworkPolicyReference{
		storage.NetworkPolicyReference_builder{
			Namespace: testNetworkPolicies[0].GetNamespace(),
			Name:      testNetworkPolicies[0].GetName(),
		}.Build(),
		storage.NetworkPolicyReference_builder{
			Namespace: testNetworkPolicies[1].GetNamespace(),
			Name:      testNetworkPolicies[1].GetName(),
		}.Build(),
		storage.NetworkPolicyReference_builder{
			Namespace: testNetworkPolicies[2].GetNamespace(),
			Name:      testNetworkPolicies[2].GetName(),
		}.Build(),
		storage.NetworkPolicyReference_builder{
			Namespace: testNetworkPolicies[3].GetNamespace(),
			Name:      testNetworkPolicies[3].GetName(),
		}.Build(),
	})
}

func sortPolicies(policies []*storage.NetworkPolicy) {
	for _, policy := range policies {
		for _, ingressRule := range policy.GetSpec().GetIngress() {
			sort.Slice(ingressRule.GetFrom(), func(i, j int) bool {
				return protocompat.MarshalTextString(ingressRule.GetFrom()[i]) < protocompat.MarshalTextString(ingressRule.GetFrom()[j])
			})
		}
		sort.Slice(policy.GetSpec().GetIngress(), func(i, j int) bool {
			return protocompat.MarshalTextString(policy.GetSpec().GetIngress()[i]) < protocompat.MarshalTextString(policy.GetSpec().GetIngress()[j])
		})
	}
	sort.Slice(policies, func(i, j int) bool {
		return protocompat.MarshalTextString(policies[i]) < protocompat.MarshalTextString(policies[j])
	})
}

func (s *generatorTestSuite) TestGenerate() {
	now := time.Now().UTC()
	ts := protoconv.ConvertTimeToTimestampOrNow(&now)
	req := &v1.GenerateNetworkPoliciesRequest{}
	req.SetClusterId("mycluster")
	req.SetDeleteExisting(v1.GenerateNetworkPoliciesRequest_NONE)
	req.SetNetworkDataSince(ts)

	ctxHasDeploymentsAccessMatcher := sacTestutils.ContextWithAccess(sac.ScopeSuffix{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
		sac.ResourceScopeKey(resources.Deployment.Resource),
		sac.ClusterScopeKey("mycluster"),
	})

	s.mockDeployments.EXPECT().SearchRawDeployments(ctxHasDeploymentsAccessMatcher, gomock.Any()).Return(
		[]*storage.Deployment{
			storage.Deployment_builder{
				Id:        "depA",
				Name:      "depA",
				Namespace: "ns1",
				PodLabels: map[string]string{"depID": "A"},
				LabelSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "A"},
				}.Build(),
			}.Build(),
			storage.Deployment_builder{
				Id:        "depB",
				Name:      "depB",
				Namespace: "ns1",
				PodLabels: map[string]string{"depID": "B"},
				LabelSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "B"},
				}.Build(),
			}.Build(),
			storage.Deployment_builder{
				Id:        "depC",
				Name:      "depC",
				Namespace: "ns1",
				PodLabels: map[string]string{"depID": "C"},
				LabelSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "C"},
				}.Build(),
			}.Build(),
			storage.Deployment_builder{
				Id:        "depD",
				Name:      "depD",
				Namespace: "ns2",
				PodLabels: map[string]string{"depID": "D"},
				LabelSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "D"},
				}.Build(),
			}.Build(),
		}, nil)

	nm := &storage.NamespaceMetadata{}
	nm.SetId("1")
	nm.SetName("ns1")
	nm.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "ns1",
	})
	nm2 := &storage.NamespaceMetadata{}
	nm2.SetId("2")
	nm2.SetName("ns2")
	nm2.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "ns2",
	})
	s.mockNamespaceStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return(
		[]*storage.NamespaceMetadata{
			nm,
			nm2,
		}, nil)

	clusterIDMatcher := testutils.PredMatcher("check cluster ID", func(clusterID string) bool { return clusterID == "mycluster" })
	s.mocksNetworkPolicies.EXPECT().GetNetworkPolicies(s.hasReadCtx, clusterIDMatcher, "").Return(
		[]*storage.NetworkPolicy{
			storage.NetworkPolicy_builder{
				Id:        "np1",
				ClusterId: "mycluster",
				Namespace: "ns1",
				Spec: storage.NetworkPolicySpec_builder{
					PodSelector: storage.LabelSelector_builder{
						MatchLabels: map[string]string{"depID": "A"},
					}.Build(),
					PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				}.Build(),
			}.Build(),
			storage.NetworkPolicy_builder{
				Id:        "np2",
				ClusterId: "mycluster",
				Namespace: "ns1",
				Spec: storage.NetworkPolicySpec_builder{
					PodSelector: storage.LabelSelector_builder{
						MatchLabels: map[string]string{"depID": "B"},
					}.Build(),
					PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
				}.Build(),
			}.Build(),
		}, nil)

	mockFlowStore := nfDSMocks.NewMockFlowDataStore(s.mockCtrl)

	ctxHasNetworkFlowAccessMatcher := sacTestutils.ContextWithAccess(
		sac.ScopeSuffix{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
			sac.ResourceScopeKey(resources.NetworkGraph.Resource),
			sac.ClusterScopeKey("mycluster"),
		})

	mockFlowStore.EXPECT().GetMatchingFlows(ctxHasNetworkFlowAccessMatcher, gomock.Any(), gomock.Eq(&now)).Return(
		[]*storage.NetworkFlow{
			storage.NetworkFlow_builder{
				Props: storage.NetworkFlowProperties_builder{
					SrcEntity: storage.NetworkEntityInfo_builder{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depA",
					}.Build(),
					DstEntity: storage.NetworkEntityInfo_builder{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depB",
					}.Build(),
				}.Build(),
			}.Build(),
			storage.NetworkFlow_builder{
				Props: storage.NetworkFlowProperties_builder{
					SrcEntity: storage.NetworkEntityInfo_builder{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depA",
					}.Build(),
					DstEntity: storage.NetworkEntityInfo_builder{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depC",
					}.Build(),
				}.Build(),
			}.Build(),
			storage.NetworkFlow_builder{
				Props: storage.NetworkFlowProperties_builder{
					SrcEntity: storage.NetworkEntityInfo_builder{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depC",
					}.Build(),
					DstEntity: storage.NetworkEntityInfo_builder{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depB",
					}.Build(),
				}.Build(),
			}.Build(),
			storage.NetworkFlow_builder{
				Props: storage.NetworkFlowProperties_builder{
					SrcEntity: storage.NetworkEntityInfo_builder{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depD",
					}.Build(),
					DstEntity: storage.NetworkEntityInfo_builder{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depB",
					}.Build(),
				}.Build(),
			}.Build(),
			storage.NetworkFlow_builder{
				Props: storage.NetworkFlowProperties_builder{
					SrcEntity: storage.NetworkEntityInfo_builder{
						Type: storage.NetworkEntityInfo_INTERNET,
					}.Build(),
					DstEntity: storage.NetworkEntityInfo_builder{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "depC",
					}.Build(),
				}.Build(),
			}.Build(),
		}, &now, nil)

	s.mockNetTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), gomock.Any()).Return(nil)
	s.mockNetTreeMgr.EXPECT().GetDefaultNetworkTree(gomock.Any()).Return(nil)
	s.mockGlobalFlowDataStore.EXPECT().GetFlowStore(gomock.Any(), gomock.Eq("mycluster")).Return(mockFlowStore, nil)

	generatedPolicies, toDelete, err := s.generator.Generate(s.hasReadCtx, req)
	s.NoError(err)
	s.Empty(toDelete)

	// canonicalize policies, strip out uninteresting fields
	for _, policy := range generatedPolicies {
		s.Equal("true", policy.GetLabels()[generatedNetworkPolicyLabel])
		policy.SetLabels(nil)
		s.Equal(networkPolicyAPIVersion, policy.GetApiVersion())
		policy.SetApiVersion("")
	}

	sortPolicies(generatedPolicies)

	expectedPolicies := []*storage.NetworkPolicy{
		// No policy for depA as there already is an existing policy
		storage.NetworkPolicy_builder{
			Name:      "stackrox-generated-depB",
			Namespace: "ns1",
			Spec: storage.NetworkPolicySpec_builder{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "B"},
				}.Build(),
				Ingress: []*storage.NetworkPolicyIngressRule{
					storage.NetworkPolicyIngressRule_builder{
						From: []*storage.NetworkPolicyPeer{
							storage.NetworkPolicyPeer_builder{
								PodSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{"depID": "A"},
								}.Build(),
							}.Build(),
							storage.NetworkPolicyPeer_builder{
								PodSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{"depID": "C"},
								}.Build(),
							}.Build(),
							storage.NetworkPolicyPeer_builder{
								NamespaceSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{namespaces.NamespaceNameLabel: "ns2"},
								}.Build(),
								PodSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{"depID": "D"},
								}.Build(),
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		}.Build(),
		storage.NetworkPolicy_builder{
			Name:      "stackrox-generated-depC",
			Namespace: "ns1",
			Spec: storage.NetworkPolicySpec_builder{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "C"},
				}.Build(),
				Ingress: []*storage.NetworkPolicyIngressRule{
					allowAllIngress,
				},
			}.Build(),
		}.Build(),
		storage.NetworkPolicy_builder{
			Name:      "stackrox-generated-depD",
			Namespace: "ns2",
			Spec: storage.NetworkPolicySpec_builder{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "D"},
				}.Build(),
			}.Build(),
		}.Build(),
	}

	sortPolicies(expectedPolicies)

	protoassert.SlicesEqual(s.T(), expectedPolicies, generatedPolicies)
}

func depFlow(fromID, toID string) *storage.NetworkFlow {
	nei := &storage.NetworkEntityInfo{}
	nei.SetType(storage.NetworkEntityInfo_DEPLOYMENT)
	nei.SetId(toID)
	nei2 := &storage.NetworkEntityInfo{}
	nei2.SetType(storage.NetworkEntityInfo_DEPLOYMENT)
	nei2.SetId(fromID)
	nfp := &storage.NetworkFlowProperties{}
	nfp.SetSrcEntity(nei)
	nfp.SetDstEntity(nei2)
	nf := &storage.NetworkFlow{}
	nf.SetProps(nfp)
	return nf
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

	now := time.Now().UTC()
	ts := protoconv.ConvertTimeToTimestampOrNow(&now)
	req := &v1.GenerateNetworkPoliciesRequest{}
	req.SetClusterId("mycluster")
	req.SetQuery("Namespace: foo,bar,qux")
	req.SetDeleteExisting(v1.GenerateNetworkPoliciesRequest_NONE)
	req.SetNetworkDataSince(ts)

	ctxHasAllDeploymentsAccessMatcher := sacTestutils.ContextWithAccess(sac.ScopeSuffix{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
		sac.ResourceScopeKey(resources.Deployment.Resource),
		sac.ClusterScopeKey("mycluster"),
	})

	s.mockDeployments.EXPECT().SearchRawDeployments(gomock.Not(ctxHasAllDeploymentsAccessMatcher), gomock.Any()).Return(
		[]*storage.Deployment{
			storage.Deployment_builder{
				Id:        "depA",
				Name:      "depA",
				Namespace: "foo",
				PodLabels: map[string]string{"depID": "A"},
				LabelSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "A"},
				}.Build(),
			}.Build(),
			storage.Deployment_builder{
				Id:        "depB",
				Name:      "depB",
				Namespace: "foo",
				PodLabels: map[string]string{"depID": "B"},
				LabelSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "B"},
				}.Build(),
			}.Build(),
			storage.Deployment_builder{
				Id:        "depC",
				Name:      "depC",
				Namespace: "foo",
				PodLabels: map[string]string{"depID": "C"},
				LabelSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "C"},
				}.Build(),
			}.Build(),
			storage.Deployment_builder{
				Id:        "depD",
				Name:      "depD",
				Namespace: "bar",
				PodLabels: map[string]string{"depID": "D"},
				LabelSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "D"},
				}.Build(),
			}.Build(),
			storage.Deployment_builder{
				Id:        "depF",
				Name:      "depF",
				Namespace: "qux",
				PodLabels: map[string]string{"depID": "F"},
				LabelSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "F"},
				}.Build(),
			}.Build(),
			storage.Deployment_builder{
				Id:        "depG",
				Name:      "depG",
				Namespace: "qux",
				PodLabels: map[string]string{"depID": "G"},
				LabelSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "G"},
				}.Build(),
			}.Build(),
		}, nil)

	ctxHasAllNamespaceAccessMatcher := sacTestutils.ContextWithAccess(sac.ScopeSuffix{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
		sac.ResourceScopeKey(resources.Namespace.Resource),
		sac.ClusterScopeKey("mycluster"),
	})

	nm := &storage.NamespaceMetadata{}
	nm.SetId("1")
	nm.SetName("foo")
	nm.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "foo",
	})
	nm2 := &storage.NamespaceMetadata{}
	nm2.SetId("2")
	nm2.SetName("bar")
	nm2.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "bar",
	})
	nm3 := &storage.NamespaceMetadata{}
	nm3.SetId("3")
	nm3.SetName("baz")
	nm3.SetLabels(map[string]string{
		namespaces.NamespaceNameLabel: "baz",
	})
	s.mockNamespaceStore.EXPECT().SearchNamespaces(gomock.Not(ctxHasAllNamespaceAccessMatcher), gomock.Any()).Return(
		[]*storage.NamespaceMetadata{
			nm,
			nm2,
			nm3,
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

	mockFlowStore.EXPECT().GetMatchingFlows(ctxHasClusterWideNetworkFlowAccessMatcher, gomock.Any(), gomock.Eq(&now)).Return(
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
		}, &now, nil)

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
			storage.Deployment_builder{
				Id:        "depD",
				Name:      "depD",
				Namespace: "bar",
				PodLabels: map[string]string{"depID": "D"},
				LabelSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "D"},
				}.Build(),
			}.Build(),
			storage.Deployment_builder{
				Id:        "depE",
				Name:      "depE",
				Namespace: "baz",
				PodLabels: map[string]string{"depID": "E"},
				LabelSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "E"},
				}.Build(),
			}.Build(),
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
		policy.SetLabels(nil)
		s.Equal(networkPolicyAPIVersion, policy.GetApiVersion())
		policy.SetApiVersion("")
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
		storage.NetworkPolicy_builder{
			Name:      "stackrox-generated-depA",
			Namespace: "foo",
			Spec: storage.NetworkPolicySpec_builder{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "A"},
				}.Build(),
				Ingress: []*storage.NetworkPolicyIngressRule{
					storage.NetworkPolicyIngressRule_builder{
						From: []*storage.NetworkPolicyPeer{
							storage.NetworkPolicyPeer_builder{
								PodSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{"depID": "B"},
								}.Build(),
							}.Build(),
							storage.NetworkPolicyPeer_builder{
								NamespaceSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{namespaces.NamespaceNameLabel: "bar"},
								}.Build(),
								PodSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{"depID": "D"},
								}.Build(),
							}.Build(),
							storage.NetworkPolicyPeer_builder{
								NamespaceSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{namespaces.NamespaceNameLabel: "baz"},
								}.Build(),
								PodSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{"depID": "E"},
								}.Build(),
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		}.Build(),
		storage.NetworkPolicy_builder{
			Name:      "stackrox-generated-depB",
			Namespace: "foo",
			Spec: storage.NetworkPolicySpec_builder{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "B"},
				}.Build(),
				Ingress: []*storage.NetworkPolicyIngressRule{
					allowAllPodsAllNS,
				},
			}.Build(),
		}.Build(),
		storage.NetworkPolicy_builder{
			Name:      "stackrox-generated-depC",
			Namespace: "foo",
			Spec: storage.NetworkPolicySpec_builder{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "C"},
				}.Build(),
				Ingress: []*storage.NetworkPolicyIngressRule{
					storage.NetworkPolicyIngressRule_builder{
						From: []*storage.NetworkPolicyPeer{
							storage.NetworkPolicyPeer_builder{
								PodSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{
										"depID": "A",
									},
								}.Build(),
							}.Build(),
							storage.NetworkPolicyPeer_builder{
								NamespaceSelector: &storage.LabelSelector{},
								PodSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{
										"depID": "F",
									},
								}.Build(),
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		}.Build(),
		storage.NetworkPolicy_builder{
			Name:      "stackrox-generated-depF",
			Namespace: "qux",
			Spec: storage.NetworkPolicySpec_builder{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "F"},
				}.Build(),
				Ingress: []*storage.NetworkPolicyIngressRule{
					storage.NetworkPolicyIngressRule_builder{
						From: []*storage.NetworkPolicyPeer{
							storage.NetworkPolicyPeer_builder{
								NamespaceSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{
										namespaces.NamespaceNameLabel: "foo",
									},
								}.Build(),
								PodSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{
										"depID": "C",
									},
								}.Build(),
							}.Build(),
							storage.NetworkPolicyPeer_builder{
								NamespaceSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{
										namespaces.NamespaceNameLabel: "bar",
									},
								}.Build(),
								PodSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{
										"depID": "D",
									},
								}.Build(),
							}.Build(),
							storage.NetworkPolicyPeer_builder{
								PodSelector: storage.LabelSelector_builder{
									MatchLabels: map[string]string{
										"depID": "G",
									},
								}.Build(),
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		}.Build(),
		storage.NetworkPolicy_builder{
			Name:      "stackrox-generated-depG",
			Namespace: "qux",
			Spec: storage.NetworkPolicySpec_builder{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: storage.LabelSelector_builder{
					MatchLabels: map[string]string{"depID": "G"},
				}.Build(),
				Ingress: nil,
			}.Build(),
		}.Build(),
	}

	sortPolicies(expectedPolicies)
	protoassert.SlicesEqual(s.T(), expectedPolicies, generatedPolicies)
}
