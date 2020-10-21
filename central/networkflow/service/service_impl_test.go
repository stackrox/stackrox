package service

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	dDSMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/networkflow"
	graphConfigDSMocks "github.com/stackrox/rox/central/networkflow/config/datastore/mocks"
	entityMocks "github.com/stackrox/rox/central/networkflow/datastore/entities/mocks"
	nfDSMocks "github.com/stackrox/rox/central/networkflow/datastore/mocks"
	npDSMocks "github.com/stackrox/rox/central/networkpolicies/graph/mocks"
	"github.com/stackrox/rox/central/role/resources"
	connMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/networkgraph"
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

	clusters      *clusterDSMocks.MockDataStore
	entities      *entityMocks.MockEntityDataStore
	deployments   *dDSMocks.MockDataStore
	flows         *nfDSMocks.MockClusterDataStore
	graphConfig   *graphConfigDSMocks.MockDataStore
	sensorConnMgr *connMocks.MockManager
	evaluator     *npDSMocks.MockEvaluator
	tested        *serviceImpl

	mockCtrl *gomock.Controller
}

func (s *NetworkGraphServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.clusters = clusterDSMocks.NewMockDataStore(s.mockCtrl)
	s.deployments = dDSMocks.NewMockDataStore(s.mockCtrl)
	s.entities = entityMocks.NewMockEntityDataStore(s.mockCtrl)
	s.flows = nfDSMocks.NewMockClusterDataStore(s.mockCtrl)
	s.graphConfig = graphConfigDSMocks.NewMockDataStore(s.mockCtrl)
	s.sensorConnMgr = connMocks.NewMockManager(s.mockCtrl)
	s.evaluator = npDSMocks.NewMockEvaluator(s.mockCtrl)

	s.tested = newService(s.flows, s.entities, s.deployments, s.clusters, s.graphConfig, s.sensorConnMgr)
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
	// Namespace bar:
	// - depD has incoming flows from depA and depE, and external source es3.
	// - depE has incoming flows from depD and depB
	// Namespace baz:
	// - depF has incoming flows from depB, and external source es3
	// Namespace far (invisible):
	// - depQ (invisible) has incoming flows from external source es1 and es3.
	// External Sources:
	// - es1 has incoming flow from deployments depA and depD.
	// - es3 has incoming floe from deployment depD and external source es1.
	// EXPECT:
	//   - all flows within namespace foo
	//   - flows between depD and depA, and depE and depB
	//   - incoming flow for depB from a masked deployment
	//   - flows es1 - depA, es2 - depA

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.OneStepSCC{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS): sac.OneStepSCC{
			sac.ResourceScopeKey(resources.Deployment.Resource): sac.AllowFixedScopes(
				sac.ClusterScopeKeys("mycluster"),
				sac.NamespaceScopeKeys("foo", "bar", "baz"),
			),
			sac.ResourceScopeKey(resources.NetworkGraph.Resource): sac.AllowFixedScopes(
				sac.ClusterScopeKeys("mycluster"),
				sac.NamespaceScopeKeys("foo", "baz", "far"),
			),
		},
	})

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

	if features.NetworkGraphExternalSrcs.Enabled() {
		s.entities.EXPECT().GetAllEntitiesForCluster(ctx, gomock.Any()).Return([]*storage.NetworkEntity{
			{
				Info: &storage.NetworkEntityInfo{
					Id:   "es1",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				},
			},
			{
				Info: &storage.NetworkEntityInfo{
					Id:   "es2",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				},
			},
			{
				Info: &storage.NetworkEntityInfo{
					Id:   "es3",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				},
			},
		}, nil)
	}

	mockFlowStore := nfDSMocks.NewMockFlowDataStore(s.mockCtrl)

	ctxHasClusterWideNetworkFlowAccessMatcher := sacTestutils.ContextWithAccess(
		sac.ScopeSuffix{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
			sac.ResourceScopeKey(resources.NetworkGraph.Resource),
			sac.ClusterScopeKey("mycluster"),
		})

	s.flows.EXPECT().GetFlowStore(ctxHasClusterWideNetworkFlowAccessMatcher, "mycluster").Return(mockFlowStore, nil)

	mockFlowStore.EXPECT().GetMatchingFlows(ctxHasClusterWideNetworkFlowAccessMatcher, gomock.Any(), gomock.Eq(ts)).DoAndReturn(
		func(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, _ *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error) {
			var flows []*storage.NetworkFlow
			if features.NetworkGraphExternalSrcs.Enabled() {
				flows = []*storage.NetworkFlow{depFlow("depA", "depB"),
					depFlow("depA", "depD"),
					depFlow("depA", "depE"),
					depFlow("depA", "depX"),
					depFlow("depA", "depY"),
					depFlow("depA", "depZ"),
					anyFlow("depA", storage.NetworkEntityInfo_DEPLOYMENT, "es1", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					anyFlow("depA", storage.NetworkEntityInfo_DEPLOYMENT, "es2", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					depFlow("depB", "depA"),
					depFlow("depB", "depX"),
					depFlow("depB", "depW"),
					depFlow("depC", "depA"),
					depFlow("depC", "depW"),
					anyFlow("depC", storage.NetworkEntityInfo_DEPLOYMENT, "es4", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					depFlow("depD", "depA"),
					depFlow("depD", "depE"),
					depFlow("depD", "depZ"),
					anyFlow("depD", storage.NetworkEntityInfo_DEPLOYMENT, "es3", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					depFlow("depE", "depD"),
					depFlow("depE", "depX"),
					depFlow("depE", "depB"),
					depFlow("depF", "depB"),
					anyFlow("depF", storage.NetworkEntityInfo_DEPLOYMENT, "es3", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					anyFlow("depF", storage.NetworkEntityInfo_DEPLOYMENT, "es5", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					anyFlow("depQ", storage.NetworkEntityInfo_DEPLOYMENT, "es1", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					anyFlow("depQ", storage.NetworkEntityInfo_DEPLOYMENT, "es3", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					anyFlow("depX", storage.NetworkEntityInfo_DEPLOYMENT, "es3", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					anyFlow("depX", storage.NetworkEntityInfo_DEPLOYMENT, networkgraph.InternetExternalSourceID, storage.NetworkEntityInfo_INTERNET),
					anyFlow("es1", storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depA", storage.NetworkEntityInfo_DEPLOYMENT),
					anyFlow("es1", storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depD", storage.NetworkEntityInfo_DEPLOYMENT),
					anyFlow("es2", storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depD", storage.NetworkEntityInfo_DEPLOYMENT),
					anyFlow("es3", storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depD", storage.NetworkEntityInfo_DEPLOYMENT),
					anyFlow("es3", storage.NetworkEntityInfo_EXTERNAL_SOURCE, "es1", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				}
			} else {
				flows = []*storage.NetworkFlow{depFlow("depA", "depB"),
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
				}
			}
			return networkflow.FilterFlowsByPredicate(flows, pred), *types.TimestampNow(), nil
		})

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

	var expected []string
	if features.NetworkGraphExternalSrcs.Enabled() {
		expected = []string{
			"foo/depA <- foo/depB",
			"foo/depA <- bar/depD",
			"foo/depA <- bar/depE",
			"foo/depA <- masked namespace #1/masked deployment #1",
			"foo/depA <- masked namespace #1/masked deployment #2",
			"foo/depA <- es1",
			"foo/depA <- es2",
			"foo/depB <- foo/depA",
			"foo/depB <- masked namespace #1/masked deployment #1",
			"foo/depB <- masked namespace #2/masked deployment #3",
			"foo/depC <- foo/depA",
			"foo/depC <- masked namespace #2/masked deployment #3",
			"foo/depC <- " + networkgraph.InternetExternalSourceID,
			"bar/depD <- foo/depA",
			"bar/depE <- foo/depB",
			"es1 <- foo/depA",
		}
	} else {
		expected = []string{
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

	s.deployments.EXPECT().SearchListDeployments(ctxHasAllDeploymentsAccessMatcher, gomock.Any()).Return(relevantDeployments, nil)

	if features.NetworkGraphExternalSrcs.Enabled() {
		s.entities.EXPECT().GetAllEntitiesForCluster(ctx, "mycluster").Return([]*storage.NetworkEntity{
			{
				Info: &storage.NetworkEntityInfo{
					Id:   "es1",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				},
			},
			{
				Info: &storage.NetworkEntityInfo{
					Id:   "es2",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				},
			},
			{
				Info: &storage.NetworkEntityInfo{
					Id:   "es3",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				},
			},
		}, nil)
	}

	mockFlowStore := nfDSMocks.NewMockFlowDataStore(s.mockCtrl)

	ctxHasClusterWideNetworkFlowAccessMatcher := sacTestutils.ContextWithAccess(
		sac.ScopeSuffix{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
			sac.ResourceScopeKey(resources.NetworkGraph.Resource),
			sac.ClusterScopeKey("mycluster"),
		})

	mockFlowStore.EXPECT().GetMatchingFlows(ctxHasClusterWideNetworkFlowAccessMatcher, gomock.Any(), gomock.Eq(ts)).DoAndReturn(
		func(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, _ *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error) {
			var flows []*storage.NetworkFlow
			if features.NetworkGraphExternalSrcs.Enabled() {
				flows = []*storage.NetworkFlow{
					depFlow("depA", "depB"),
					depFlow("depA", "depD"),
					depFlow("depA", "depE"),
					depFlow("depA", "depX"),
					depFlow("depA", "depY"),
					depFlow("depA", "depZ"),
					listenFlow("depA", 8443),
					anyFlow("depA", storage.NetworkEntityInfo_DEPLOYMENT, "es1", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					anyFlow("depA", storage.NetworkEntityInfo_DEPLOYMENT, "es2", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					depFlow("depB", "depA"),
					depFlow("depB", "depX"),
					depFlow("depB", "depW"),
					depFlow("depC", "depA"),
					depFlow("depC", "depW"),
					anyFlow("depC", storage.NetworkEntityInfo_DEPLOYMENT, "es4", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					depFlow("depD", "depA"),
					depFlow("depD", "depE"),
					depFlow("depD", "depZ"),
					listenFlow("depD", 53),
					listenFlow("depD", 8080),
					anyFlow("depD", storage.NetworkEntityInfo_DEPLOYMENT, "es3", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					depFlow("depE", "depD"),
					depFlow("depE", "depX"),
					depFlow("depE", "depB"),
					depFlow("depF", "depB"),
					anyFlow("depF", storage.NetworkEntityInfo_DEPLOYMENT, "es3", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					anyFlow("depQ", storage.NetworkEntityInfo_DEPLOYMENT, "es1", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					anyFlow("depQ", storage.NetworkEntityInfo_DEPLOYMENT, "es3", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					anyFlow("depX", storage.NetworkEntityInfo_DEPLOYMENT, "es3", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
					anyFlow("es1", storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depA", storage.NetworkEntityInfo_DEPLOYMENT),
					anyFlow("es1", storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depD", storage.NetworkEntityInfo_DEPLOYMENT),
					anyFlow("es2", storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depD", storage.NetworkEntityInfo_DEPLOYMENT),
					anyFlow("es3", storage.NetworkEntityInfo_EXTERNAL_SOURCE, "depD", storage.NetworkEntityInfo_DEPLOYMENT),
					anyFlow("es3", storage.NetworkEntityInfo_EXTERNAL_SOURCE, "es1", storage.NetworkEntityInfo_EXTERNAL_SOURCE),
				}
			} else {
				flows = []*storage.NetworkFlow{
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
				}
			}
			return networkflow.FilterFlowsByPredicate(flows, pred), *types.TimestampNow(), nil
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

	var expected []string
	if features.NetworkGraphExternalSrcs.Enabled() {
		expected = []string{
			"foo/depA <- foo/depB",
			"foo/depA <- bar/depD",
			"foo/depA <- bar/depE",
			"foo/depA <- es1",
			"foo/depA <- es2",
			"foo/depB <- foo/depA",
			"foo/depC <- foo/depA",
			"foo/depC <- " + networkgraph.InternetExternalSourceID,
			"bar/depD <- foo/depA",
			"bar/depD <- bar/depE",
			"bar/depD <- es3",
			"bar/depE <- foo/depB",
			"bar/depE <- bar/depD",
			"es1 <- foo/depA",
			"es1 <- bar/depD",
			"es2 <- bar/depD",
			"es3 <- bar/depD",
		}
	} else {
		expected = []string{
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
	}
	sort.Strings(expected)
	sort.Strings(flowStrings)
	s.Equal(expected, flowStrings)
}

func (s *NetworkGraphServiceTestSuite) TestCreateExternalNetworkEntity() {
	if !features.NetworkGraphExternalSrcs.Enabled() {
		s.T().Skip()
	}

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

	s.entities.EXPECT().UpsertExternalNetworkEntity(ctx, gomock.Any()).Return(nil)
	s.clusters.EXPECT().Exists(gomock.Any(), "c1").Return(true, nil)
	pushSig := concurrency.NewSignal()
	s.sensorConnMgr.EXPECT().PushExternalNetworkEntitiesToSensor(ctx, "c1").DoAndReturn(
		func(ctx context.Context, clusterID string) error {
			s.Equal("c1", clusterID)
			pushSig.Signal()
			return nil
		})

	_, err = s.tested.CreateExternalNetworkEntity(ctx, request)
	s.NoError(err)
	s.True(concurrency.WaitWithTimeout(&pushSig, time.Second*1))

	// Cluster not found-no flows upserted
	s.clusters.EXPECT().Exists(gomock.Any(), "c1").Return(false, nil)
	_, err = s.tested.CreateExternalNetworkEntity(ctx, request)
	s.Error(err)
}

func (s *NetworkGraphServiceTestSuite) TestDeleteExternalNetworkEntity() {
	if !features.NetworkGraphExternalSrcs.Enabled() {
		s.T().Skip()
	}

	ctx := sac.WithAllAccess(context.Background())

	id, _ := sac.NewClusterScopeResourceID("c1", "id")
	request := &v1.ResourceByID{
		Id: id.ToString(),
	}

	s.entities.EXPECT().DeleteExternalNetworkEntity(ctx, gomock.Any()).Return(nil)
	pushSig := concurrency.NewSignal()
	s.sensorConnMgr.EXPECT().PushExternalNetworkEntitiesToSensor(ctx, "c1").DoAndReturn(
		func(ctx context.Context, clusterID string) error {
			s.Equal("c1", clusterID)
			pushSig.Signal()
			return nil
		})

	_, err := s.tested.DeleteExternalNetworkEntity(ctx, request)
	s.NoError(err)
	s.True(concurrency.WaitWithTimeout(&pushSig, time.Second*1))
}

func (s *NetworkGraphServiceTestSuite) TestPatchExternalNetworkEntity() {
	if !features.NetworkGraphExternalSrcs.Enabled() {
		s.T().Skip()
	}

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
	s.entities.EXPECT().UpsertExternalNetworkEntity(ctx, gomock.Any()).Return(nil)
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
	if !features.NetworkGraphExternalSrcs.Enabled() {
		s.T().Skip()
	}

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
