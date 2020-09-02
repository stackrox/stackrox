package service

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	dDSMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/networkflow"
	nfDSMocks "github.com/stackrox/rox/central/networkflow/datastore/mocks"
	npDSMocks "github.com/stackrox/rox/central/networkpolicies/graph/mocks"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	sacTestutils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestNetworkGraph(t *testing.T) {
	suite.Run(t, new(NetworkGraphServiceTestSuite))
}

type NetworkGraphServiceTestSuite struct {
	suite.Suite
	deployments *dDSMocks.MockDataStore
	flows       *nfDSMocks.MockClusterDataStore
	evaluator   *npDSMocks.MockEvaluator
	tested      *serviceImpl

	mockCtrl *gomock.Controller
}

func (s *NetworkGraphServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.deployments = dDSMocks.NewMockDataStore(s.mockCtrl)
	s.flows = nfDSMocks.NewMockClusterDataStore(s.mockCtrl)

	s.evaluator = npDSMocks.NewMockEvaluator(s.mockCtrl)

	s.tested = newService(s.flows, s.deployments)
}

func (s *NetworkGraphServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkGraphServiceTestSuite) TestFailsIfClusterIsNotSet() {
	request := &v1.NetworkGraphRequest{}
	_, err := s.tested.GetNetworkGraph(context.Background(), request)
	s.Error(err, "expected graph generation to fail since no cluster is specified")
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
	// Third namespace baz is visible but not selected
	// User has no network flow access in namespace bar
	// Namespace foo has deployments:
	// - depA has incoming flows from depB, depD, depE, deployment depX and depZ in a secret namespace,
	//   and deployment depY that was recently deleted
	// - depB has incoming flows from depA and deployment depX in a secret namespace, and depW in another secret namespace
	// - depC has incoming flows from depA and depW
	// Namespace bar:
	// - depD has incoming flows from depA and depE
	// - depE has incoming flows from depD and depB
	// Namespace baz:
	// - depF has incoming flows from depB
	// EXPECT:
	//   - all flows within namespace foo
	//   - flows between depD and depA, and depE and depB
	//   - incoming flow for depB from a masked deployment

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.OneStepSCC{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS): sac.OneStepSCC{
			sac.ResourceScopeKey(resources.Deployment.Resource): sac.AllowFixedScopes(
				sac.ClusterScopeKeys("mycluster"),
				sac.NamespaceScopeKeys("foo", "bar", "baz"),
			),
			sac.ResourceScopeKey(resources.NetworkGraph.Resource): sac.AllowFixedScopes(
				sac.ClusterScopeKeys("mycluster"),
				sac.NamespaceScopeKeys("foo", "baz"),
			),
		},
	})

	ts := types.TimestampNow()
	req := &v1.NetworkGraphRequest{
		ClusterId: "mycluster",
		Query:     "Namespace: foo,bar",
		Since:     ts,
	}

	ctxHasAllDeploymentsAccessMatcher := sacTestutils.ContextWithAccess(sac.ScopeSuffix{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
		sac.ResourceScopeKey(resources.Deployment.Resource),
		sac.ClusterScopeKey("mycluster"),
	})

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

	mockFlowStore := nfDSMocks.NewMockFlowDataStore(s.mockCtrl)

	ctxHasClusterWideNetworkFlowAccessMatcher := sacTestutils.ContextWithAccess(
		sac.ScopeSuffix{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
			sac.ResourceScopeKey(resources.NetworkGraph.Resource),
			sac.ClusterScopeKey("mycluster"),
		})

	mockFlowStore.EXPECT().GetMatchingFlows(ctxHasClusterWideNetworkFlowAccessMatcher, gomock.Any(), gomock.Eq(ts)).DoAndReturn(
		func(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, _ *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error) {
			return networkflow.FilterFlowsByPredicate([]*storage.NetworkFlow{
				depFlow("depA", "depB"),
				depFlow("depA", "depD"),
				depFlow("depA", "depE"),
				depFlow("depA", "depX"),
				depFlow("depA", "depY"),
				depFlow("depA", "depZ"),
				depFlow("depB", "depA"),
				depFlow("depB", "depX"),
				depFlow("depB", "depW"),
				depFlow("depC", "depA"),
				depFlow("depC", "depW"),
				depFlow("depD", "depA"),
				depFlow("depD", "depE"),
				depFlow("depD", "depZ"),
				depFlow("depE", "depD"),
				depFlow("depE", "depX"),
				depFlow("depE", "depB"),
				depFlow("depF", "depB"),
			}, pred), *types.TimestampNow(), nil
		})

	s.flows.EXPECT().GetFlowStore(ctxHasClusterWideNetworkFlowAccessMatcher, "mycluster").Return(mockFlowStore, nil)

	s.deployments.EXPECT().Search(gomock.Not(ctxHasAllDeploymentsAccessMatcher), gomock.Any()).Return(
		[]search.Result{
			{ID: "depA"},
			{ID: "depB"},
			{ID: "depC"},
			{ID: "depD"},
			{ID: "depE"},
			{ID: "depF"},
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
			// depY was deleted
		}, nil)

	graph, err := s.tested.GetNetworkGraph(ctx, req)
	s.Require().NotNil(graph)
	s.Require().NoError(err)

	var flowStrings []string
	for _, node := range graph.GetNodes() {
		for succIdx := range node.GetOutEdges() {
			succ := graph.GetNodes()[succIdx]
			srcDeploy, dstDeploy := node.GetEntity().GetDeployment(), succ.GetEntity().GetDeployment()
			flowStrings = append(flowStrings, fmt.Sprintf("%s/%s <- %s/%s", dstDeploy.GetNamespace(), dstDeploy.GetName(), srcDeploy.GetNamespace(), srcDeploy.GetName()))
		}
	}

	expected := []string{
		"foo/depA <- foo/depB",
		"foo/depA <- bar/depD",
		"foo/depA <- bar/depE",
		"foo/depA <- masked namespace #1/masked deployment #1",
		"foo/depA <- masked namespace #1/masked deployment #2",
		"foo/depB <- foo/depA",
		"foo/depB <- masked namespace #1/masked deployment #1",
		"foo/depB <- masked namespace #2/masked deployment #3",
		"foo/depC <- foo/depA",
		"foo/depC <- masked namespace #2/masked deployment #3",
		"bar/depD <- foo/depA",
		"bar/depE <- foo/depB",
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
	// - depA has incoming flows from depB, depD, depE, deployment depX and depZ in a secret namespace,
	//   and deployment depY that was recently deleted
	// - depB has incoming flows from depA and deployment depX in a secret namespace, and depW in another secret namespace
	// - depC has incoming flows from depA and depW
	// Namespace bar:
	// - depD has incoming flows from depA and depE
	// - depE has incoming flows from depD and depB
	// Namespace baz:
	// - depF has incoming flows from depB
	// EXPECT:
	//   - all flows within namespace foo
	//   - flows between depD and depA, and depE and depB
	//   - incoming flow for depB from a masked deployment

	ctx := sac.WithAllAccess(context.Background())

	ts := types.TimestampNow()
	req := &v1.NetworkGraphRequest{
		ClusterId: "mycluster",
		Query:     "Namespace: foo,bar",
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

	s.deployments.EXPECT().SearchListDeployments(ctxHasAllDeploymentsAccessMatcher, gomock.Any()).Return(relevantDeployments, nil)

	mockFlowStore := nfDSMocks.NewMockFlowDataStore(s.mockCtrl)

	ctxHasClusterWideNetworkFlowAccessMatcher := sacTestutils.ContextWithAccess(
		sac.ScopeSuffix{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
			sac.ResourceScopeKey(resources.NetworkGraph.Resource),
			sac.ClusterScopeKey("mycluster"),
		})

	mockFlowStore.EXPECT().GetMatchingFlows(ctxHasClusterWideNetworkFlowAccessMatcher, gomock.Any(), gomock.Eq(ts)).DoAndReturn(
		func(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, _ *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error) {
			return networkflow.FilterFlowsByPredicate([]*storage.NetworkFlow{
				depFlow("depA", "depB"),
				depFlow("depA", "depD"),
				depFlow("depA", "depE"),
				depFlow("depA", "depX"),
				depFlow("depA", "depY"),
				depFlow("depA", "depZ"),
				listenFlow("depA", 8443),
				depFlow("depB", "depA"),
				depFlow("depB", "depX"),
				depFlow("depB", "depW"),
				depFlow("depC", "depA"),
				depFlow("depC", "depW"),
				depFlow("depD", "depA"),
				depFlow("depD", "depE"),
				depFlow("depD", "depZ"),
				listenFlow("depD", 53),
				listenFlow("depD", 8080),
				depFlow("depE", "depD"),
				depFlow("depE", "depX"),
				depFlow("depE", "depB"),
				depFlow("depF", "depB"),
			}, pred), *types.TimestampNow(), nil
		})

	s.flows.EXPECT().GetFlowStore(ctxHasClusterWideNetworkFlowAccessMatcher, "mycluster").Return(mockFlowStore, nil)

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
		srcDeploy := node.GetEntity().GetDeployment()
		s.NotNil(srcDeploy)
		for succIdx := range node.GetOutEdges() {
			succ := graph.GetNodes()[succIdx]
			dstDeploy := succ.GetEntity().GetDeployment()
			flowStrings = append(flowStrings, fmt.Sprintf("%s/%s <- %s/%s", dstDeploy.GetNamespace(), dstDeploy.GetName(), srcDeploy.GetNamespace(), srcDeploy.GetName()))
		}

		s.ElementsMatch(srcDeploy.GetListenPorts(), expectedListenPorts[node.GetEntity().GetId()])
	}

	expected := []string{
		"foo/depA <- foo/depB",
		"foo/depA <- bar/depD",
		"foo/depA <- bar/depE",
		"foo/depB <- foo/depA",
		"foo/depC <- foo/depA",
		"bar/depD <- foo/depA",
		"bar/depD <- bar/depE",
		"bar/depE <- foo/depB",
		"bar/depE <- bar/depD",
	}
	sort.Strings(expected)
	sort.Strings(flowStrings)
	s.Equal(expected, flowStrings)
}
