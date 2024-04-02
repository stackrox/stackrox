package generator

import (
	"context"
	"fmt"
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

const (
	clusterID = "mycluster"
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

var (
	deploymentA = &storage.Deployment{
		Id:        "depA",
		Name:      "depA",
		Namespace: "ns1",
		ClusterId: clusterID,
		PodLabels: map[string]string{"depID": "A"},
		LabelSelector: &storage.LabelSelector{
			MatchLabels: map[string]string{"depID": "A"},
		},
	}
	deploymentB = &storage.Deployment{
		Id:        "depB",
		Name:      "depB",
		Namespace: "ns1",
		ClusterId: clusterID,
		PodLabels: map[string]string{"depID": "B"},
		LabelSelector: &storage.LabelSelector{
			MatchLabels: map[string]string{"depID": "B"},
		},
	}
	deploymentC = &storage.Deployment{
		Id:        "depC",
		Name:      "depC",
		Namespace: "ns1",
		ClusterId: clusterID,
		PodLabels: map[string]string{"depID": "C"},
		LabelSelector: &storage.LabelSelector{
			MatchLabels: map[string]string{"depID": "C"},
		},
	}
	deploymentD = &storage.Deployment{
		Id:        "depD",
		Name:      "depD",
		Namespace: "ns2",
		ClusterId: clusterID,
		PodLabels: map[string]string{"depID": "D"},
		LabelSelector: &storage.LabelSelector{
			MatchLabels: map[string]string{"depID": "D"},
		},
	}
	deploymentE = &storage.Deployment{
		Id:        "depE",
		Name:      "depE",
		Namespace: "baz",
		ClusterId: clusterID,
		PodLabels: map[string]string{"depID": "E"},
		LabelSelector: &storage.LabelSelector{
			MatchLabels: map[string]string{"depID": "E"},
		},
	}
	deploymentF = &storage.Deployment{
		Id:        "depF",
		Name:      "depF",
		Namespace: "qux",
		ClusterId: clusterID,
		PodLabels: map[string]string{"depID": "F"},
		LabelSelector: &storage.LabelSelector{
			MatchLabels: map[string]string{"depID": "F"},
		},
	}
	deploymentG = &storage.Deployment{
		Id:        "depG",
		Name:      "depG",
		Namespace: "qux",
		ClusterId: clusterID,
		PodLabels: map[string]string{"depID": "G"},
		LabelSelector: &storage.LabelSelector{
			MatchLabels: map[string]string{"depID": "G"},
		},
	}
	deploymentH = &storage.Deployment{
		Id:        "depH",
		Name:      "depH",
		Namespace: "baz",
		ClusterId: clusterID,
		PodLabels: map[string]string{"depID": "H"},
		LabelSelector: &storage.LabelSelector{
			MatchLabels: map[string]string{"depID": "H"},
		},
	}
	deploymentX = &storage.Deployment{
		Id:        "depX",
		Name:      "depX",
		Namespace: "baz",
		ClusterId: clusterID,
		PodLabels: map[string]string{"depID": "X"},
		LabelSelector: &storage.LabelSelector{
			MatchLabels: map[string]string{"depID": "X"},
		},
	}

	testNetworkPolicies = []*storage.NetworkPolicy{
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
)

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
				return protocompat.MarshalTextString(ingressRule.From[i]) < protocompat.MarshalTextString(ingressRule.From[j])
			})
		}
		sort.Slice(policy.Spec.Ingress, func(i, j int) bool {
			return protocompat.MarshalTextString(policy.Spec.Ingress[i]) < protocompat.MarshalTextString(policy.Spec.Ingress[j])
		})
	}
	sort.Slice(policies, func(i, j int) bool {
		return protocompat.MarshalTextString(policies[i]) < protocompat.MarshalTextString(policies[j])
	})
}

func generateExternalNetworkPeer(ingress bool, port uint32, protocol storage.L4Protocol) *storage.NetworkBaselinePeer {
	return &storage.NetworkBaselinePeer{
		Properties: []*storage.NetworkBaselineConnectionProperties{
			{
				Ingress:  ingress,
				Port:     port,
				Protocol: protocol,
			},
		},
		Entity: &storage.NetworkEntity{
			Info: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Id:   "id",
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Name: "external",
						Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
							Cidr: "192.0.2.0/24",
						},
						Default: true,
					},
				},
			},
		},
	}
}

func generateDeploymentNetworkPeer(dstID string, dstName string, dstNamespace string, clusterID string, ingress bool, port uint32, protocol storage.L4Protocol) *storage.NetworkBaselinePeer {
	return &storage.NetworkBaselinePeer{
		Properties: []*storage.NetworkBaselineConnectionProperties{
			{
				Ingress:  ingress,
				Port:     port,
				Protocol: protocol,
			},
		},
		Entity: &storage.NetworkEntity{
			Scope: &storage.NetworkEntity_Scope{
				ClusterId: clusterID,
			},
			Info: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   dstID,
				Desc: &storage.NetworkEntityInfo_Deployment_{
					Deployment: &storage.NetworkEntityInfo_Deployment{
						Name:      dstName,
						Namespace: dstNamespace,
						Cluster:   clusterID,
					},
				},
			},
		},
	}
}

func deploymentsToMap(deployments ...*storage.Deployment) map[string]*storage.Deployment {
	ret := make(map[string]*storage.Deployment)
	for _, d := range deployments {
		ret[d.GetId()] = d
	}
	return ret
}

func deploymentsToSlice(deployments ...*storage.Deployment) []*storage.Deployment {
	ret := append([]*storage.Deployment{}, deployments...)
	return ret
}

func generateBaselineMapFromDeployments(deployments ...*storage.Deployment) map[string]*storage.NetworkBaseline {
	ret := make(map[string]*storage.NetworkBaseline)
	for _, d := range deployments {
		ret[d.GetId()] = &storage.NetworkBaseline{
			DeploymentId:   d.GetId(),
			DeploymentName: d.GetName(),
			ClusterId:      d.GetClusterId(),
			Namespace:      d.GetNamespace(),
			Peers:          []*storage.NetworkBaselinePeer{},
			ForbiddenPeers: []*storage.NetworkBaselinePeer{},
		}
	}
	return ret
}

var (
	namespaceMetadata = []*storage.NamespaceMetadata{
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
	}
	existingNetworkPolicies = []*storage.NetworkPolicy{
		{
			Id:        "np1",
			ClusterId: clusterID,
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
			ClusterId: clusterID,
			Namespace: "ns1",
			Spec: &storage.NetworkPolicySpec{
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "B"},
				},
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE},
			},
		},
	}
	expectedPolicies = []*storage.NetworkPolicy{
		// No policy for depA as there already is an existing policy
		{
			Name:      "stackrox-generated-depB",
			Namespace: "ns1",
			ClusterId: clusterID,
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
			ClusterId: clusterID,
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
			ClusterId: clusterID,
			Spec: &storage.NetworkPolicySpec{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "D"},
				},
			},
		},
	}
)

func (s *generatorTestSuite) TestGenerateWithBaselines() {
	now := time.Now().UTC()
	ts := protoconv.ConvertTimeToTimestampOrNow(&now)
	req := &v1.GenerateNetworkPoliciesRequest{
		ClusterId:        clusterID,
		DeleteExisting:   v1.GenerateNetworkPoliciesRequest_NONE,
		NetworkDataSince: ts,
	}

	ctxHasDeploymentsAccessMatcher := sacTestutils.ContextWithAccess(sac.ScopeSuffix{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
		sac.ResourceScopeKey(resources.Deployment.Resource),
		sac.ClusterScopeKey(clusterID),
	})

	deployments := deploymentsToSlice(deploymentA, deploymentB, deploymentC, deploymentD)
	deploymentsMap := deploymentsToMap(deployments...)
	baselines := generateBaselineMapFromDeployments(deployments...)
	// egress flows are not taken into consideration, we add them in the test to illustrative purposes
	baselines[deploymentA.GetId()].Peers = []*storage.NetworkBaselinePeer{
		generateDeploymentNetworkPeer("depB", "depB", "ns1", clusterID, false, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
		generateDeploymentNetworkPeer("depC", "depC", "ns1", clusterID, false, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
	}
	baselines[deploymentB.GetId()].Peers = []*storage.NetworkBaselinePeer{
		generateDeploymentNetworkPeer("depA", "depA", "ns1", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
		generateDeploymentNetworkPeer("depC", "depC", "ns1", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
		generateDeploymentNetworkPeer("depD", "depD", "ns2", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
	}
	baselines[deploymentC.GetId()].Peers = []*storage.NetworkBaselinePeer{
		generateDeploymentNetworkPeer("depB", "depB", "ns1", clusterID, false, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
		generateDeploymentNetworkPeer("depA", "depA", "ns1", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
		generateExternalNetworkPeer(true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
	}
	baselines[deploymentD.GetId()].Peers = []*storage.NetworkBaselinePeer{
		generateDeploymentNetworkPeer("depB", "depB", "ns1", clusterID, false, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
	}
	s.mockDeployments.EXPECT().SearchRawDeployments(ctxHasDeploymentsAccessMatcher, gomock.Any()).Return(deployments, nil)

	s.mockNetworkBaselineStore.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).
		Times(len(deployments)).
		DoAndReturn(func(_ any, deploymentID string) (*storage.NetworkBaseline, bool, error) {
			baseline, ok := baselines[deploymentID]
			s.Require().True(ok)
			return baseline, true, nil
		})
	s.mockDeployments.EXPECT().GetDeployment(gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(func(_ any, deploymentID string) (*storage.Deployment, bool, error) {
			deployment, ok := deploymentsMap[deploymentID]
			s.Require().True(ok)
			return deployment, true, nil
		})

	s.mockNamespaceStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return(namespaceMetadata, nil)

	clusterIDMatcher := testutils.PredMatcher("check cluster ID", func(id string) bool { return id == clusterID })
	s.mocksNetworkPolicies.EXPECT().GetNetworkPolicies(s.hasReadCtx, clusterIDMatcher, "").Return(existingNetworkPolicies, nil)

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
	sortPolicies(expectedPolicies)

	s.Equal(expectedPolicies, generatedPolicies)
}

func depFlow(toID, fromID string) *storage.NetworkFlow {
	return &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   fromID,
			},
			DstEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   toID,
			},
		},
	}
}

func (s *generatorTestSuite) TestGenerateWithMissingBaselines() {
	now := time.Now().UTC()
	ts := protoconv.ConvertTimeToTimestampOrNow(&now)
	req := &v1.GenerateNetworkPoliciesRequest{
		ClusterId:        clusterID,
		DeleteExisting:   v1.GenerateNetworkPoliciesRequest_NONE,
		NetworkDataSince: ts,
	}

	ctxHasDeploymentsAccessMatcher := sacTestutils.ContextWithAccess(sac.ScopeSuffix{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
		sac.ResourceScopeKey(resources.Deployment.Resource),
		sac.ClusterScopeKey(clusterID),
	})

	deployments := deploymentsToSlice(deploymentA, deploymentB, deploymentC, deploymentD)
	s.mockDeployments.EXPECT().SearchRawDeployments(ctxHasDeploymentsAccessMatcher, gomock.Any()).Return(deployments, nil)

	s.mockNetworkBaselineStore.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).
		Times(len(deployments)).
		DoAndReturn(func(_ any, _ string) (*storage.NetworkBaseline, bool, error) {
			return nil, false, nil
		})

	s.mockNamespaceStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return(namespaceMetadata, nil)

	mockFlowStore := nfDSMocks.NewMockFlowDataStore(s.mockCtrl)

	ctxHasNetworkFlowAccessMatcher := sacTestutils.ContextWithAccess(
		sac.ScopeSuffix{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
			sac.ResourceScopeKey(resources.NetworkGraph.Resource),
			sac.ClusterScopeKey(clusterID),
		})

	mockFlowStore.EXPECT().GetMatchingFlows(ctxHasNetworkFlowAccessMatcher, gomock.Any(), gomock.Eq(&now)).Return(
		[]*storage.NetworkFlow{
			depFlow(deploymentB.GetId(), deploymentA.GetId()),
			depFlow(deploymentC.GetId(), deploymentA.GetId()),
			depFlow(deploymentB.GetId(), deploymentC.GetId()),
			depFlow(deploymentB.GetId(), deploymentD.GetId()),
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
		}, &now, nil)

	s.mockNetTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), gomock.Any()).Return(nil)
	s.mockNetTreeMgr.EXPECT().GetDefaultNetworkTree(gomock.Any()).Return(nil)
	s.mockGlobalFlowDataStore.EXPECT().GetFlowStore(gomock.Any(), gomock.Eq(clusterID)).Return(mockFlowStore, nil)

	clusterIDMatcher := testutils.PredMatcher("check cluster ID", func(id string) bool { return id == clusterID })
	s.mocksNetworkPolicies.EXPECT().GetNetworkPolicies(s.hasReadCtx, clusterIDMatcher, "").Return(existingNetworkPolicies, nil)

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
	sortPolicies(expectedPolicies)

	s.Equal(expectedPolicies, generatedPolicies)
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
	// - depH has incoming flows from depA and a deployment without access, depX. Its baseline is missing
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
							clusterID: {Namespaces: []string{"foo", "bar", "baz", "qux"}},
						},
					},
					resources.NetworkGraph.Resource: &sac.TestResourceScope{
						Clusters: map[string]*sac.TestClusterScope{
							clusterID: {Namespaces: []string{"foo", "baz", "qux"}},
						},
					},
					resources.Namespace.Resource: &sac.TestResourceScope{
						Clusters: map[string]*sac.TestClusterScope{
							clusterID: {Namespaces: []string{"foo", "bar", "baz"}},
						},
					},
				},
			}))

	now := time.Now().UTC()
	ts := protoconv.ConvertTimeToTimestampOrNow(&now)
	req := &v1.GenerateNetworkPoliciesRequest{
		ClusterId:        clusterID,
		Query:            "Namespace: foo,bar,qux",
		DeleteExisting:   v1.GenerateNetworkPoliciesRequest_NONE,
		NetworkDataSince: ts,
	}

	ctxHasAllDeploymentsAccessMatcher := sacTestutils.ContextWithAccess(sac.ScopeSuffix{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
		sac.ResourceScopeKey(resources.Deployment.Resource),
		sac.ClusterScopeKey(clusterID),
	})
	// Cloning the deployments A, B, C, and D as we are changing the namespaces for this test
	depA := deploymentA.Clone()
	depA.Namespace = "foo"
	depB := deploymentB.Clone()
	depB.Namespace = "foo"
	depC := deploymentC.Clone()
	depC.Namespace = "foo"
	depD := deploymentD.Clone()
	depD.Namespace = "bar"
	deployments := deploymentsToSlice(depA, depB, depC, depD, deploymentH, deploymentF, deploymentG)
	deploymentsMap := deploymentsToMap(depA, depB, depC, depD, deploymentF, deploymentG, deploymentE)
	baselines := generateBaselineMapFromDeployments(depA, depB, depC, depD, deploymentF, deploymentG)
	baselines[depA.GetId()].Peers = []*storage.NetworkBaselinePeer{
		generateDeploymentNetworkPeer("depB", "depB", "foo", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
		generateDeploymentNetworkPeer("depE", "depE", "baz", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
		generateDeploymentNetworkPeer("depD", "depD", "bar", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
		// Deployment Y was deleted
		generateDeploymentNetworkPeer("depY", "depY", "deleted", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
	}
	baselines[depB.GetId()].Peers = []*storage.NetworkBaselinePeer{
		generateDeploymentNetworkPeer("depA", "depA", "foo", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
		generateDeploymentNetworkPeer("depX", "depX", "baz", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
	}
	baselines[depC.GetId()].Peers = []*storage.NetworkBaselinePeer{
		generateDeploymentNetworkPeer("depF", "depF", "qux", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
		generateDeploymentNetworkPeer("depA", "depA", "foo", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
	}
	baselines[deploymentF.GetId()].Peers = []*storage.NetworkBaselinePeer{
		generateDeploymentNetworkPeer("depC", "depC", "foo", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
		generateDeploymentNetworkPeer("depG", "depG", "qux", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
		generateDeploymentNetworkPeer("depD", "depD", "bar", clusterID, true, 80, storage.L4Protocol_L4_PROTOCOL_TCP),
	}
	s.mockDeployments.EXPECT().GetDeployment(gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(func(ctx context.Context, deploymentID string) (*storage.Deployment, bool, error) {
			val, ok := ctx.Value(sac.GetGlobalAccessScopeContextKey(s.T())).(sac.ScopeCheckerCore)
			if !ok {
				return nil, false, nil
			}
			// If the call is made with not enough privileges
			if !val.Allowed() {
				deployment, ok := deploymentsMap[deploymentID]
				if !ok {
					return nil, false, nil
				}
				return deployment, true, nil
			}
			// Privileged request returns the deployment X
			if deploymentID == deploymentX.GetId() {
				return deploymentX, true, nil
			}
			return nil, false, nil
		})
	s.mockDeployments.EXPECT().SearchRawDeployments(gomock.Not(ctxHasAllDeploymentsAccessMatcher), gomock.Any()).Return(deployments, nil)

	ctxHasAllNamespaceAccessMatcher := sacTestutils.ContextWithAccess(sac.ScopeSuffix{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
		sac.ResourceScopeKey(resources.Namespace.Resource),
		sac.ClusterScopeKey(clusterID),
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
	mockFlowStore := nfDSMocks.NewMockFlowDataStore(s.mockCtrl)

	ctxHasClusterWideNetworkFlowAccessMatcher := sacTestutils.ContextWithAccess(
		sac.ScopeSuffix{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
			sac.ResourceScopeKey(resources.NetworkGraph.Resource),
			sac.ClusterScopeKey(clusterID),
		})

	mockFlowStore.EXPECT().GetMatchingFlows(ctxHasClusterWideNetworkFlowAccessMatcher, gomock.Any(), gomock.Eq(&now)).Return(
		[]*storage.NetworkFlow{
			depFlow(deploymentH.GetId(), depA.GetId()),
			depFlow(deploymentH.GetId(), deploymentX.GetId()),
		}, &now, nil)

	s.mockNetTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), gomock.Any()).Return(nil)
	s.mockNetTreeMgr.EXPECT().GetDefaultNetworkTree(gomock.Any()).Return(nil)
	s.mockGlobalFlowDataStore.EXPECT().GetFlowStore(gomock.Any(), gomock.Eq(clusterID)).Return(mockFlowStore, nil)

	// Expect a query for looking up deployments that were not selected as part of the initial query
	// (visible or invisible).
	s.mockDeployments.EXPECT().GetDeployments(
		gomock.Not(ctxHasAllDeploymentsAccessMatcher),
		// depX is part of the query since it was eliminated as irrelevant before.
		testutils.AssertionMatcher(assert.ElementsMatch, []string{depA.GetId(), deploymentX.GetId()})).Return(
		[]*storage.Deployment{depA}, nil,
	)
	// Expect a query with elevated privileges for looking up deployments that we are still missing info about
	// (either deleted or invisible to the user).
	s.mockDeployments.EXPECT().Search(ctxHasAllDeploymentsAccessMatcher, gomock.Any()).Return(
		[]search.Result{
			{
				ID: deploymentX.GetId(),
			},
			// depY was deleted!
		}, nil)
	// Assume no existing network policies.
	s.mocksNetworkPolicies.EXPECT().GetNetworkPolicies(gomock.Any(), clusterID, "").Return(nil, nil)

	s.mockNetworkBaselineStore.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(func(_ any, deploymentID string) (*storage.NetworkBaseline, bool, error) {
			s.T().Log("Baseline for deployment ", deploymentID)
			baseline, ok := baselines[deploymentID]
			if !ok {
				s.T().Log("Baseline for deployment ", deploymentID, " not found")
				return nil, false, nil
			}
			s.T().Log("Baseline for deployment ", deploymentID, " found")
			return baseline, true, nil
		})

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
	// - netpol for depH allowing all cluster traffic
	expectedNetworkPolicies := []*storage.NetworkPolicy{
		{
			Name:      "stackrox-generated-depA",
			Namespace: "foo",
			ClusterId: clusterID,
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
			ClusterId: clusterID,
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
			ClusterId: clusterID,
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
			ClusterId: clusterID,
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
			ClusterId: clusterID,
			Spec: &storage.NetworkPolicySpec{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "G"},
				},
				Ingress: nil,
			},
		},
		{
			Name:      "stackrox-generated-depH",
			Namespace: "baz",
			ClusterId: clusterID,
			Spec: &storage.NetworkPolicySpec{
				PolicyTypes: []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE},
				PodSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{"depID": "H"},
				},
				Ingress: []*storage.NetworkPolicyIngressRule{
					allowAllPodsAllNS,
				},
			},
		},
	}

	sortPolicies(expectedNetworkPolicies)
	s.Equal(expectedNetworkPolicies, generatedPolicies)
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
