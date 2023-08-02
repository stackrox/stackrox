package service

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/gogo/protobuf/types"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	dDSMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	graphConfigDSMocks "github.com/stackrox/rox/central/networkgraph/config/datastore/mocks"
	entityMocks "github.com/stackrox/rox/central/networkgraph/entity/datastore/mocks"
	networkTreeMocks "github.com/stackrox/rox/central/networkgraph/entity/networktree/mocks"
	nfDSMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	networkPolicyMocks "github.com/stackrox/rox/central/networkpolicies/datastore/mocks"
	npDSMocks "github.com/stackrox/rox/central/networkpolicies/graph/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	sacTestutils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestNetworkGraph(t *testing.T) {
	suite.Run(t, new(NetworkGraphServiceTestSuite))
}

type NetworkGraphServiceTestSuite struct {
	suite.Suite

	clusters       *clusterDSMocks.MockDataStore
	entities       *entityMocks.MockEntityDataStore
	deployments    *dDSMocks.MockDataStore
	flows          *nfDSMocks.MockClusterDataStore
	graphConfig    *graphConfigDSMocks.MockDataStore
	networkTreeMgr *networkTreeMocks.MockManager
	policies       *networkPolicyMocks.MockDataStore

	evaluator *npDSMocks.MockEvaluator
	tested    *serviceImpl

	mockCtrl *gomock.Controller
}

func (s *NetworkGraphServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.clusters = clusterDSMocks.NewMockDataStore(s.mockCtrl)
	s.deployments = dDSMocks.NewMockDataStore(s.mockCtrl)
	s.entities = entityMocks.NewMockEntityDataStore(s.mockCtrl)
	s.flows = nfDSMocks.NewMockClusterDataStore(s.mockCtrl)
	s.graphConfig = graphConfigDSMocks.NewMockDataStore(s.mockCtrl)
	s.evaluator = npDSMocks.NewMockEvaluator(s.mockCtrl)
	s.networkTreeMgr = networkTreeMocks.NewMockManager(s.mockCtrl)
	s.policies = networkPolicyMocks.NewMockDataStore(s.mockCtrl)

	s.tested = newService(s.flows, s.entities, s.networkTreeMgr, s.deployments, s.clusters, s.policies, s.graphConfig)
}

func (s *NetworkGraphServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkGraphServiceTestSuite) TestFailsIfClusterIsNotSet() {
	request := &v1.NetworkGraphRequest{}
	_, err := s.tested.GetNetworkGraph(context.Background(), request)
	s.Error(err, "expected graph generation to fail since no cluster is specified")
}

func flowAsString(src, dst *storage.NetworkEntityInfo) string {
	var srcString string
	var dstString string
	if src.GetDeployment() != nil {
		srcString = fmt.Sprintf("%s/%s", src.GetDeployment().GetNamespace(), src.GetDeployment().GetName())
	} else {
		srcString = src.GetId()
	}

	if dst.GetDeployment() != nil {
		dstString = fmt.Sprintf("%s/%s", dst.GetDeployment().GetNamespace(), dst.GetDeployment().GetName())
	} else {
		dstString = dst.GetId()
	}
	return fmt.Sprintf("%s <- %s", dstString, srcString)
}

func anyFlow(toID string, toType storage.NetworkEntityInfo_Type, fromID string, fromType storage.NetworkEntityInfo_Type) *storage.NetworkFlow {
	return &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity: &storage.NetworkEntityInfo{
				Type: fromType,
				Id:   fromID,
			},
			DstEntity: &storage.NetworkEntityInfo{
				Type: toType,
				Id:   toID,
			},
		},
	}
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

func listenFlow(depID string, port uint32) *storage.NetworkFlow {
	return &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   depID,
			},
			DstEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_LISTEN_ENDPOINT,
			},
			DstPort:    port,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
	}
}

func (s *NetworkGraphServiceTestSuite) TestGenerateNetworkGraphWithAllAccess() {
	s.testGenerateNetworkGraphAllAccess(false)
}

func (s *NetworkGraphServiceTestSuite) TestGenerateNetworkGraphWithAllAccessAndListenPorts() {
	s.testGenerateNetworkGraphAllAccess(true)
}

func (s *NetworkGraphServiceTestSuite) TestGenerateNetworkGraphWithSAC() {
	// Test setup:
	// Query selects namespace foo and bar (visible)
	// Namespace baz is visible but not selected
	// Namespace far is not visible but selected
	// User has no network flow access in namespace bar
	// Namespace foo has deployments:
	// - depA has incoming flows from depB, depD, depE, deployment depX and depZ in a secret namespace,
	//   and deployment depY that was recently deleted, and external sources es1 and es2.
	// - depB has incoming flows from depA and deployment depX in a secret namespace, and depW in another secret namespace
	// - depC has incoming flows from depA and depW, and deleted external source es4.
	// - depG and depH are orchestrator components.
	// Namespace bar:
	// - depD has incoming flows from depA and depE, and external source es3.
	// - depE has incoming flows from depD and depB
	// Namespace baz:
	// - depF has incoming flows from depB, and external source es3
	// Namespace far (invisible):
	// - depQ (invisible) has incoming flows from external source es1 and es3.
	// External Sources:
	// - es1 has incoming flow from deployments depA and depD.
	// - es3 has incoming flow from deployment depD and external source es1.
	// EXPECT:
	//   - all flows within namespace foo
	//   - flows to/from namespace foo and bar
	//   - flows between deployments in namespace foo and bar and masked deployments depX, depZ, and depW
	//   - flows es1 - depA, es2 - depA

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.TestScopeCheckerCoreFromFullScopeMap(s.T(),
			sac.TestScopeMap{
				storage.Access_READ_ACCESS: {
					resources.Deployment.Resource: &sac.TestResourceScope{
						Clusters: map[string]*sac.TestClusterScope{
							"mycluster": {Namespaces: []string{"foo", "bar", "baz"}},
						},
					},
					resources.NetworkGraph.Resource: &sac.TestResourceScope{
						Clusters: map[string]*sac.TestClusterScope{
							"mycluster": {Namespaces: []string{"foo", "baz", "far"}},
						},
					},
				},
			}))

	ts := types.TimestampNow()
	req := &v1.NetworkGraphRequest{
		ClusterId: "mycluster",
		Query:     "Namespace: foo,bar,far",
		Scope: &v1.NetworkGraphScope{
			Query: "Orchestrator Component:false",
		},
		Since: ts,
	}

	ctxHasAllDeploymentsAccessMatcher := sacTestutils.ContextWithAccess(sac.ScopeSuffix{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
		sac.ResourceScopeKey(resources.Deployment.Resource),
		sac.ClusterScopeKey("mycluster"),
	})

	s.deployments.EXPECT().Count(gomock.Any(), gomock.Any()).Return(5, nil)
	s.deployments.EXPECT().SearchListDeployments(gomock.Not(ctxHasAllDeploymentsAccessMatcher), gomock.Any()).Return(
		[]*storage.ListDeployment{
			{
				Id:        "depA",
				Name:      "depA",
				Namespace: "foo",
			},
			{
				Id:        "depB",
				Name:      "depB",
				Namespace: "foo",
			},
			{
				Id:        "depC",
				Name:      "depC",
				Namespace: "foo",
			},
			{
				Id:        "depD",
				Name:      "depD",
				Namespace: "bar",
			},
			{
				Id:        "depE",
				Name:      "depE",
				Namespace: "bar",
			},
		}, nil)

	es1aID, _ := externalsrcs.NewClusterScopedID("mycluster", "35.187.144.0/20")
	es1bID, _ := externalsrcs.NewClusterScopedID("mycluster", "35.187.144.0/16")
	es1cID, _ := externalsrcs.NewClusterScopedID("mycluster", "35.187.144.0/8")
	es2ID, _ := externalsrcs.NewClusterScopedID("mycluster", "35.187.144.0/23")
	es3ID, _ := externalsrcs.NewClusterScopedID("mycluster", "36.188.144.0/16")
	es4ID, _ := externalsrcs.NewClusterScopedID("mycluster", "10.10.10.10/8")
	es5ID, _ := externalsrcs.NewClusterScopedID("mycluster", "36.188.144.0/30")

	es1a := testutils.GetExtSrcNetworkEntityInfo(es1aID.String(), "net1", "35.187.144.0/20", false)
	es1b := testutils.GetExtSrcNetworkEntityInfo(es1bID.String(), "net1", "35.187.144.0/16", false)
	es1c := testutils.GetExtSrcNetworkEntityInfo(es1cID.String(), "net1", "35.187.144.0/8", false)
	es2 := testutils.GetExtSrcNetworkEntityInfo(es2ID.String(), "2", "35.187.144.0/23", false)
	es3 := testutils.GetExtSrcNetworkEntityInfo(es3ID.String(), "3", "36.188.144.0/16", false)

	networkTree, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{es1a, es1b, es1c, es2, es3})
	s.NoError(err)

	mockFlowStore := nfDSMocks.NewMockFlowDataStore(s.mockCtrl)

	ctxHasClusterWideNetworkFlowAccessMatcher := sacTestutils.ContextWithAccess(
		sac.ScopeSuffix{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
			sac.ResourceScopeKey(resources.NetworkGraph.Resource),
			sac.ClusterScopeKey("mycluster"),
		})

	s.flows.EXPECT().GetFlowStore(ctxHasClusterWideNetworkFlowAccessMatcher, "mycluster").Return(mockFlowStore, nil)

	mockFlowStore.EXPECT().GetMatchingFlows(ctxHasClusterWideNetworkFlowAccessMatcher, gomock.Any(), gomock.Eq(ts)).DoAndReturn(
		func(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, _ *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error) {
			flows := []*storage.NetworkFlow{depFlow("depA", "depB"),
				depFlow("depA", "depD"),
				depFlow("depA", "depE"),
				depFlow("depA", "depG"),
				depFlow("depA", "depH"),
				depFlow("depA", "depX"),
				depFlow("depA", "depY"),
				depFlow("depA", "depZ"),
				anyFlow("depA", storage.NetworkEntityInfo_DEPLOYMENT, es1aID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depA", storage.NetworkEntityInfo_DEPLOYMENT, es1bID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depA", storage.NetworkEntityInfo_DEPLOYMENT, es1cID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depA", storage.NetworkEntityInfo_DEPLOYMENT, es2ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depA", storage.NetworkEntityInfo_DEPLOYMENT, es5ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				depFlow("depB", "depA"),
				depFlow("depB", "depF"),
				depFlow("depB", "depG"),
				depFlow("depB", "depH"),
				depFlow("depB", "depX"),
				depFlow("depB", "depW"),
				depFlow("depC", "depA"),
				depFlow("depC", "depW"),
				anyFlow("depC", storage.NetworkEntityInfo_DEPLOYMENT, es4ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				depFlow("depD", "depA"),
				depFlow("depD", "depE"),
				depFlow("depD", "depZ"),
				anyFlow("depD", storage.NetworkEntityInfo_DEPLOYMENT, es3ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				depFlow("depE", "depD"),
				depFlow("depE", "depX"),
				depFlow("depE", "depB"),
				depFlow("depF", "depB"),
				depFlow("depD", "depF"),
				anyFlow("depG", storage.NetworkEntityInfo_DEPLOYMENT, es3ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depH", storage.NetworkEntityInfo_DEPLOYMENT, es3ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depF", storage.NetworkEntityInfo_DEPLOYMENT, es3ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depF", storage.NetworkEntityInfo_DEPLOYMENT, es5ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depQ", storage.NetworkEntityInfo_DEPLOYMENT, es1aID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depQ", storage.NetworkEntityInfo_DEPLOYMENT, es3ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depX", storage.NetworkEntityInfo_DEPLOYMENT, es3ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depX", storage.NetworkEntityInfo_DEPLOYMENT, networkgraph.InternetExternalSourceID, storage.NetworkEntityInfo_INTERNET),
				anyFlow(es1aID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depA", storage.NetworkEntityInfo_DEPLOYMENT),
				anyFlow(es1aID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depD", storage.NetworkEntityInfo_DEPLOYMENT),
				anyFlow(es2.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depD", storage.NetworkEntityInfo_DEPLOYMENT),
				anyFlow(es3ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depD", storage.NetworkEntityInfo_DEPLOYMENT),
				anyFlow(es3ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE, es2ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
			}
			return networkgraph.FilterFlowsByPredicate(flows, pred), types.TimestampNow(), nil
		})

	s.networkTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), gomock.Any()).Return(networkTree)
	s.networkTreeMgr.EXPECT().GetDefaultNetworkTree(gomock.Any()).Return(networkTree)

	s.deployments.EXPECT().SearchListDeployments(gomock.Not(ctxHasAllDeploymentsAccessMatcher), gomock.Any()).Return(
		[]*storage.ListDeployment{
			{
				Id:        "depA",
				Name:      "depA",
				Namespace: "foo",
			},
			{
				Id:        "depB",
				Name:      "depB",
				Namespace: "foo",
			},
			{
				Id:        "depC",
				Name:      "depC",
				Namespace: "foo",
			},
			{
				Id:        "depD",
				Name:      "depD",
				Namespace: "bar",
			},

			{
				Id:        "depE",
				Name:      "depE",
				Namespace: "bar",
			},

			{
				Id:        "depF",
				Name:      "depF",
				Namespace: "baz",
			},
		}, nil)

	s.deployments.EXPECT().SearchListDeployments(ctxHasAllDeploymentsAccessMatcher, gomock.Any()).Return(
		[]*storage.ListDeployment{
			{
				Id:        "depX",
				Name:      "depX",
				Namespace: "secretns",
			},
			{
				Id:        "depZ",
				Name:      "depZ",
				Namespace: "secretns",
			},
			{
				Id:        "depW",
				Name:      "depW",
				Namespace: "supersecretns",
			},
			{
				Id:        "depQ",
				Name:      "depQ",
				Namespace: "far",
			},
			// depY was deleted
		}, nil)

	graph, err := s.tested.GetNetworkGraph(ctx, req)
	s.Require().NotNil(graph)
	s.Require().NoError(err)

	var flowStrings []string
	for _, node := range graph.GetNodes() {
		for succIdx := range node.GetOutEdges() {
			succ := graph.GetNodes()[succIdx]
			src, dst := node.GetEntity(), succ.GetEntity()

			flowStrings = append(flowStrings, flowAsString(src, dst))
		}
	}

	expected := []string{
		"foo/depA <- foo/depB",
		"foo/depA <- bar/depD",
		"foo/depA <- bar/depE",
		"foo/depA <- masked namespace #1/masked deployment #2", // depX
		"foo/depA <- masked namespace #1/masked deployment #3", // depZ
		"foo/depA <- mycluster__net1",
		"foo/depA <- " + es2ID.String(),
		"foo/depA <- " + es3ID.String(), // non-existent es5 mapped to supernet es3
		"foo/depB <- foo/depA",
		"foo/depB <- baz/depF",
		"foo/depB <- masked namespace #1/masked deployment #2", // depX
		"foo/depB <- masked namespace #2/masked deployment #1", // depW
		"foo/depC <- foo/depA",
		"foo/depC <- masked namespace #2/masked deployment #1", // depW
		"foo/depC <- " + networkgraph.InternetExternalSourceID,
		"bar/depD <- foo/depA",
		"bar/depE <- foo/depB",
		"baz/depF <- foo/depB",
		"mycluster__net1 <- foo/depA",
	}
	sort.Strings(expected)
	sort.Strings(flowStrings)
	s.Equal(expected, flowStrings)
}

func (s *NetworkGraphServiceTestSuite) testGenerateNetworkGraphAllAccess(withListenPorts bool) {
	// Test setup:
	// Query selects namespace foo and bar (visible)
	// Third namespace baz is visible but not selected
	// User has no network flow access in namespace bar
	// Namespace foo has deployments:
	// - depA has incoming flows from depB, depD, depE, depX and depZ
	//   and deployment depY that was recently deleted
	// - depB has incoming flows from depA, depX, and depW
	// - depC has incoming flows from depA and depW
	// Namespace bar:
	// - depD has incoming flows from depA, depE, and depZ
	// - depE has incoming flows from depD and depB
	// Namespace baz:
	// - depF has incoming flows from depB
	// Namespace other:
	// - depX and depZ
	// Namespace otherother:
	// - depW
	// EXPECT:
	//   - all flows within namespace foo
	//   - flows to/from namespace foo and bar

	ctx := sac.WithAllAccess(context.Background())

	ts := types.TimestampNow()
	req := &v1.NetworkGraphRequest{
		ClusterId: "mycluster",
		Query:     "Namespace: foo,bar,far",
		Since:     ts,
	}

	ctxHasAllDeploymentsAccessMatcher := sacTestutils.ContextWithAccess(sac.ScopeSuffix{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
		sac.ResourceScopeKey(resources.Deployment.Resource),
		sac.ClusterScopeKey("mycluster"),
	})

	relevantDeployments := []*storage.ListDeployment{
		{
			Id:        "depA",
			Name:      "depA",
			Namespace: "foo",
		},
		{
			Id:        "depB",
			Name:      "depB",
			Namespace: "foo",
		},
		{
			Id:        "depC",
			Name:      "depC",
			Namespace: "foo",
		},
		{
			Id:        "depD",
			Name:      "depD",
			Namespace: "bar",
		},
		{
			Id:        "depE",
			Name:      "depE",
			Namespace: "bar",
		},
	}

	s.deployments.EXPECT().Count(ctxHasAllDeploymentsAccessMatcher, gomock.Any()).Return(len(relevantDeployments), nil)
	s.deployments.EXPECT().SearchListDeployments(ctxHasAllDeploymentsAccessMatcher, gomock.Any()).Return(relevantDeployments, nil)

	es1ID, _ := externalsrcs.NewClusterScopedID("mycluster", "35.187.144.0/20")
	es2ID, _ := externalsrcs.NewClusterScopedID("mycluster", "35.187.144.0/23")
	es3ID, _ := externalsrcs.NewClusterScopedID("mycluster", "36.188.144.0/16")
	es4ID, _ := externalsrcs.NewClusterScopedID("mycluster", "10.10.10.10/8")

	es1 := testutils.GetExtSrcNetworkEntityInfo(es1ID.String(), "1", "35.187.144.0/20", false)
	es2 := testutils.GetExtSrcNetworkEntityInfo(es2ID.String(), "2", "35.187.144.0/23", false)
	es3 := testutils.GetExtSrcNetworkEntityInfo(es3ID.String(), "3", "36.188.144.0/16", false)

	networkTree, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{es1, es2, es3})
	s.NoError(err)

	mockFlowStore := nfDSMocks.NewMockFlowDataStore(s.mockCtrl)

	ctxHasClusterWideNetworkFlowAccessMatcher := sacTestutils.ContextWithAccess(
		sac.ScopeSuffix{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
			sac.ResourceScopeKey(resources.NetworkGraph.Resource),
			sac.ClusterScopeKey("mycluster"),
		})

	mockFlowStore.EXPECT().GetMatchingFlows(ctxHasClusterWideNetworkFlowAccessMatcher, gomock.Any(), gomock.Eq(ts)).DoAndReturn(
		func(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, _ *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error) {
			flows := []*storage.NetworkFlow{
				depFlow("depA", "depB"),
				depFlow("depA", "depD"),
				depFlow("depA", "depE"),
				depFlow("depA", "depX"),
				depFlow("depA", "depY"),
				depFlow("depA", "depZ"),
				listenFlow("depA", 8443),
				anyFlow("depA", storage.NetworkEntityInfo_DEPLOYMENT, es1ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depA", storage.NetworkEntityInfo_DEPLOYMENT, es2ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				depFlow("depB", "depA"),
				depFlow("depB", "depX"),
				depFlow("depB", "depW"),
				depFlow("depC", "depA"),
				depFlow("depC", "depW"),
				anyFlow("depC", storage.NetworkEntityInfo_DEPLOYMENT, es4ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				depFlow("depD", "depA"),
				depFlow("depD", "depE"),
				depFlow("depD", "depZ"),
				listenFlow("depD", 53),
				listenFlow("depD", 8080),
				anyFlow("depD", storage.NetworkEntityInfo_DEPLOYMENT, es3ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				depFlow("depE", "depD"),
				depFlow("depE", "depX"),
				depFlow("depE", "depB"),
				depFlow("depF", "depB"),
				anyFlow("depF", storage.NetworkEntityInfo_DEPLOYMENT, es3ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depQ", storage.NetworkEntityInfo_DEPLOYMENT, es1ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depQ", storage.NetworkEntityInfo_DEPLOYMENT, es3ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow("depX", storage.NetworkEntityInfo_DEPLOYMENT, es3ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				anyFlow(es1ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depA", storage.NetworkEntityInfo_DEPLOYMENT),
				anyFlow(es1ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depD", storage.NetworkEntityInfo_DEPLOYMENT),
				anyFlow(es2ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depD", storage.NetworkEntityInfo_DEPLOYMENT),
				anyFlow(es3ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depD", storage.NetworkEntityInfo_DEPLOYMENT),
				anyFlow(es3ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE, es1ID.String(), storage.NetworkEntityInfo_EXTERNAL_SOURCE),
			}
			return networkgraph.FilterFlowsByPredicate(flows, pred), types.TimestampNow(), nil
		})

	s.networkTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), gomock.Any()).Return(networkTree)
	s.networkTreeMgr.EXPECT().GetDefaultNetworkTree(gomock.Any()).Return(networkTree)

	s.flows.EXPECT().GetFlowStore(ctxHasClusterWideNetworkFlowAccessMatcher, "mycluster").Return(mockFlowStore, nil)

	s.deployments.EXPECT().SearchListDeployments(ctx, gomock.Any()).Return(
		[]*storage.ListDeployment{
			{
				Id:        "depA",
				Name:      "depA",
				Namespace: "foo",
			},
			{
				Id:        "depB",
				Name:      "depB",
				Namespace: "foo",
			},
			{
				Id:        "depC",
				Name:      "depC",
				Namespace: "foo",
			},
			{
				Id:        "depD",
				Name:      "depD",
				Namespace: "bar",
			},

			{
				Id:        "depE",
				Name:      "depE",
				Namespace: "bar",
			},

			{
				Id:        "depF",
				Name:      "depF",
				Namespace: "baz",
			},
			{
				Id:        "depX",
				Name:      "depX",
				Namespace: "other",
			},
			{
				Id:        "depZ",
				Name:      "depZ",
				Namespace: "other",
			},
			{
				Id:        "depW",
				Name:      "depW",
				Namespace: "otherother",
			},
		}, nil)

	s.deployments.EXPECT().SearchListDeployments(ctxHasAllDeploymentsAccessMatcher, gomock.Any()).Return(
		[]*storage.ListDeployment{
			// depY was deleted
		}, nil)

	var expectedListenPorts map[string][]*storage.NetworkEntityInfo_Deployment_ListenPort
	if withListenPorts {
		expectedListenPorts = map[string][]*storage.NetworkEntityInfo_Deployment_ListenPort{
			"depA": {
				{Port: 8443, L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP},
			},
			"depD": {
				{Port: 53, L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP},
				{Port: 8080, L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP},
			},
		}
	}

	graph, err := s.tested.getNetworkGraph(ctx, req, withListenPorts)
	s.Require().NotNil(graph)
	s.Require().NoError(err)

	var flowStrings []string
	for _, node := range graph.GetNodes() {
		src := node.GetEntity()
		srcDeploy := src.GetDeployment()
		if !networkgraph.IsExternal(src) {
			s.NotNil(node.GetEntity().GetDeployment())
		}
		for succIdx := range node.GetOutEdges() {
			succ := graph.GetNodes()[succIdx]
			dst := succ.GetEntity()
			flowStrings = append(flowStrings, flowAsString(src, dst))
		}

		s.ElementsMatch(srcDeploy.GetListenPorts(), expectedListenPorts[node.GetEntity().GetId()])
	}

	expected := []string{
		"foo/depA <- foo/depB",
		"foo/depA <- bar/depD",
		"foo/depA <- bar/depE",
		"foo/depA <- other/depX",
		"foo/depA <- other/depZ",
		"foo/depA <- " + es1ID.String(),
		"foo/depA <- " + es2ID.String(),
		"foo/depB <- foo/depA",
		"foo/depB <- other/depX",
		"foo/depB <- otherother/depW",
		"foo/depC <- foo/depA",
		"foo/depC <- otherother/depW",
		"foo/depC <- " + networkgraph.InternetExternalSourceID,
		"bar/depD <- foo/depA",
		"bar/depD <- bar/depE",
		"bar/depD <- other/depZ",
		"bar/depD <- " + es3ID.String(),
		"bar/depE <- foo/depB",
		"bar/depE <- bar/depD",
		"bar/depE <- other/depX",
		"baz/depF <- foo/depB",
		es1ID.String() + " <- foo/depA",
		es1ID.String() + " <- bar/depD",
		es2ID.String() + " <- bar/depD",
		es3ID.String() + " <- bar/depD",
	}
	sort.Strings(expected)
	sort.Strings(flowStrings)
	s.Equal(expected, flowStrings)
}

func (s *NetworkGraphServiceTestSuite) TestCreateExternalNetworkEntity() {
	ctx := sac.WithAllAccess(context.Background())

	// Validation failure-no cluster ID provided
	request := &v1.CreateNetworkEntityRequest{
		ClusterId: "",
		Entity: &storage.NetworkEntityInfo_ExternalSource{
			Name: "cidr1",
			Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
				Cidr: "192.0.2.0/24",
			},
		},
	}
	_, err := s.tested.CreateExternalNetworkEntity(ctx, request)
	s.Error(err)

	// Valid request
	request = &v1.CreateNetworkEntityRequest{
		ClusterId: "c1",
		Entity: &storage.NetworkEntityInfo_ExternalSource{
			Name: "cidr1",
			Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
				Cidr: "192.0.2.0/24",
			},
		},
	}

	s.entities.EXPECT().CreateExternalNetworkEntity(ctx, gomock.Any(), false).Return(nil)
	s.clusters.EXPECT().Exists(gomock.Any(), "c1").Return(true, nil)
	_, err = s.tested.CreateExternalNetworkEntity(ctx, request)
	s.NoError(err)

	// Cluster not found-no flows upserted
	s.clusters.EXPECT().Exists(gomock.Any(), "c1").Return(false, nil)
	_, err = s.tested.CreateExternalNetworkEntity(ctx, request)
	s.Error(err)
}

func (s *NetworkGraphServiceTestSuite) TestDeleteExternalNetworkEntity() {
	ctx := sac.WithAllAccess(context.Background())

	id, _ := sac.NewClusterScopeResourceID("c1", "id")
	request := &v1.ResourceByID{
		Id: id.String(),
	}

	s.entities.EXPECT().GetEntity(ctx, request.GetId()).Return(&storage.NetworkEntity{}, true, nil)
	s.entities.EXPECT().DeleteExternalNetworkEntity(ctx, request.GetId()).Return(nil)
	_, err := s.tested.DeleteExternalNetworkEntity(ctx, request)
	s.NoError(err)

	s.entities.EXPECT().GetEntity(ctx, request.GetId()).Return(&storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Name: "any",
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "net",
					},
					Default: true,
				}}}}, true, nil)
	_, err = s.tested.DeleteExternalNetworkEntity(ctx, request)
	s.Error(err)
}

func (s *NetworkGraphServiceTestSuite) TestPatchExternalNetworkEntity() {
	ctx := sac.WithAllAccess(context.Background())

	// Store an entity first.
	entity := &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id: "cidr1",
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Name: "cidr1",
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "192.0.2.0/24",
					},
				},
			},
		},
	}

	// Valid request
	patch := &v1.PatchNetworkEntityRequest{
		Id:   entity.GetInfo().GetId(),
		Name: "newcidr",
	}

	s.entities.EXPECT().GetEntity(ctx, entity.GetInfo().GetId()).Return(entity, true, nil)
	s.entities.EXPECT().UpdateExternalNetworkEntity(ctx, gomock.Any(), false).Return(nil)
	actual, err := s.tested.PatchExternalNetworkEntity(ctx, patch)
	s.NoError(err)
	entity.Info.GetExternalSource().Name = "newcidr"
	s.Equal(entity, actual)

	// Not found
	s.entities.EXPECT().GetEntity(ctx, entity.GetInfo().GetId()).Return(nil, false, nil)
	actual, err = s.tested.PatchExternalNetworkEntity(ctx, patch)
	s.Error(err)
	s.Nil(actual)

	// Invalid stored entity
	s.entities.EXPECT().GetEntity(ctx, entity.GetInfo().GetId()).Return(nil, true, nil)
	actual, err = s.tested.PatchExternalNetworkEntity(ctx, patch)
	s.Error(err)
	s.Nil(actual)
}

func (s *NetworkGraphServiceTestSuite) TestNetworkGraphConfiguration() {
	ctx := sac.WithAllAccess(context.Background())

	s.graphConfig.EXPECT().GetNetworkGraphConfig(ctx).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
	_, err := s.tested.GetNetworkGraphConfig(ctx, &v1.Empty{})
	s.NoError(err)

	s.graphConfig.EXPECT().UpdateNetworkGraphConfig(ctx, &storage.NetworkGraphConfig{HideDefaultExternalSrcs: true}).Return(nil)
	_, err = s.tested.PutNetworkGraphConfig(ctx, &v1.PutNetworkGraphConfigRequest{
		Config: &storage.NetworkGraphConfig{
			HideDefaultExternalSrcs: true,
		},
	})
	s.NoError(err)
}

func (s *NetworkGraphServiceTestSuite) TestReturnErrorIfNumberOfNodesExceedsLimit() {
	testCases := map[string]struct {
		deploymentCount int
		envValue        string
		expectedMax     int
	}{
		"Default": {
			deploymentCount: 2001,
			envValue:        "",
			expectedMax:     2000,
		},
		"Specific Env value": {
			deploymentCount: 1001,
			envValue:        "1000",
			expectedMax:     1000,
		},
		"Incorrect env value shouldn't panic": {
			deploymentCount: 2001,
			envValue:        "dummy",
			expectedMax:     2000,
		},
	}

	for name, testCase := range testCases {
		s.Run(name, func() {
			if testCase.envValue != "" {
				s.T().Setenv(maxNumberOfDeploymentsInGraphEnv.EnvVar(), testCase.envValue)
			}

			s.deployments.EXPECT().Count(gomock.Any(), gomock.Any()).Return(testCase.deploymentCount, nil)

			ctx := sac.WithAllAccess(context.Background())

			ts := types.TimestampNow()
			req := &v1.NetworkGraphRequest{
				ClusterId: "mycluster",
				Query:     "Namespace: foo,bar,far",
				Since:     ts,
			}

			_, err := s.tested.GetNetworkGraph(ctx, req)
			s.Errorf(
				err,
				"number of deployments (%d) exceeds maximum allowed for Network Graph: %d",
				testCase.deploymentCount,
				testCase.expectedMax,
			)
		})
	}
}
