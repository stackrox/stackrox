package service

import (
	"context"
	"fmt"
	"testing"

	cDataStoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	dDataStoreMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	networkBaselineDSMocks "github.com/stackrox/rox/central/networkbaseline/datastore/mocks"
	graphConfigMocks "github.com/stackrox/rox/central/networkgraph/config/datastore/mocks"
	netEntityDSMocks "github.com/stackrox/rox/central/networkgraph/entity/datastore/mocks"
	netTreeMgrMocks "github.com/stackrox/rox/central/networkgraph/entity/networktree/mocks"
	npMocks "github.com/stackrox/rox/central/networkpolicies/datastore/mocks"
	npGraphMocks "github.com/stackrox/rox/central/networkpolicies/graph/mocks"
	nDataStoreMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	grpcTestutils "github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

const fakeClusterID = "FAKECLUSTERID"
const fakeDeploymentID = "FAKEDEPLOYMENTID"
const badYAML = `
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: first-policy
spec:
  podSelector: {}
  ingress: []
`
const fakeYAML1 = `
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: first-policy
  namespace: default
spec:
  podSelector: {}
  ingress: []
`
const fakeYAML2 = `
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: second-policy
  namespace: default
spec:
  podSelector: {}
  ingress: []
`
const combinedYAMLs = `---
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: first-policy
  namespace: default
spec:
  podSelector: {}
  ingress: []
---
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: second-policy
  namespace: default
spec:
  podSelector: {}
  ingress: []
`

func TestNetworkPolicyService(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

type ServiceTestSuite struct {
	suite.Suite

	requestContext   context.Context
	clusters         *cDataStoreMocks.MockDataStore
	deployments      *dDataStoreMocks.MockDataStore
	namespaces       *namespaceMocks.MockDataStore
	externalSrcs     *netEntityDSMocks.MockEntityDataStore
	graphConfig      *graphConfigMocks.MockDataStore
	networkBaselines *networkBaselineDSMocks.MockDataStore
	netTreeMgr       *netTreeMgrMocks.MockManager
	networkPolicies  *npMocks.MockDataStore
	evaluator        *npGraphMocks.MockEvaluator
	notifiers        *nDataStoreMocks.MockDataStore
	tested           Service
	mockCtrl         *gomock.Controller
}

func (suite *ServiceTestSuite) SetupTest() {
	// Since all the datastores underneath are mocked, the context of the request doesns't need any permissions.
	suite.requestContext = sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.TestScopeCheckerCoreFromFullScopeMap(
			suite.T(),
			sac.TestScopeMap{
				storage.Access_READ_ACCESS: map[permissions.Resource]*sac.TestResourceScope{
					resources.NetworkPolicy.GetResource(): {
						Clusters: nil,
						Included: true,
					},
				},
			},
		),
	)

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.networkPolicies = npMocks.NewMockDataStore(suite.mockCtrl)
	suite.namespaces = namespaceMocks.NewMockDataStore(suite.mockCtrl)
	suite.evaluator = npGraphMocks.NewMockEvaluator(suite.mockCtrl)
	suite.clusters = cDataStoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.deployments = dDataStoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.externalSrcs = netEntityDSMocks.NewMockEntityDataStore(suite.mockCtrl)
	suite.graphConfig = graphConfigMocks.NewMockDataStore(suite.mockCtrl)
	suite.networkBaselines = networkBaselineDSMocks.NewMockDataStore(suite.mockCtrl)
	suite.netTreeMgr = netTreeMgrMocks.NewMockManager(suite.mockCtrl)
	suite.notifiers = nDataStoreMocks.NewMockDataStore(suite.mockCtrl)

	suite.tested = New(suite.networkPolicies, suite.deployments, suite.externalSrcs, suite.graphConfig, suite.networkBaselines, suite.netTreeMgr,
		suite.evaluator, suite.namespaces, suite.clusters, suite.notifiers, nil, nil)
}

func (suite *ServiceTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *ServiceTestSuite) TestAuth() {
	grpcTestutils.AssertAuthzWorks(suite.T(), suite.tested)
}

func (suite *ServiceTestSuite) TestFailsIfClusterIsNotSet() {
	request := &v1.SimulateNetworkGraphRequest{}
	_, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.Error(err, "expected graph generation to fail since no cluster is specified")
}

func (suite *ServiceTestSuite) TestFailsIfClusterDoesNotExist() {
	testCtx := sac.WithGlobalAccessScopeChecker(
		suite.requestContext,
		sac.TestScopeCheckerCoreFromFullScopeMap(
			suite.T(),
			sac.TestScopeMap{
				storage.Access_READ_ACCESS: map[permissions.Resource]*sac.TestResourceScope{
					resources.NetworkPolicy.GetResource(): {
						Clusters: nil,
						Included: true,
					},
				},
			},
		),
	)

	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(false, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{}
	request.SetClusterId(fakeClusterID)
	request.SetIncludeNodeDiff(true)
	_, err := suite.tested.SimulateNetworkGraph(testCtx, request)
	suite.Error(err, "expected graph generation to fail since cluster does not exist")
}

func (suite *ServiceTestSuite) TestRejectsYamlWithoutNamespace() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Make the request to the service and check that it did not err.
	npm := &storage.NetworkPolicyModification{}
	npm.SetApplyYaml(badYAML)
	request := &v1.SimulateNetworkGraphRequest{}
	request.SetClusterId(fakeClusterID)
	request.SetModification(npm)
	request.SetIncludeNodeDiff(true)
	_, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.Error(err, "expected graph generation to fail since input yaml has no namespace")
}

func (suite *ServiceTestSuite) TestGetNetworkGraph() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	pols := make([]*storage.NetworkPolicy, 0)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(pols, nil)
	ngc := &storage.NetworkGraphConfig{}
	ngc.SetHideDefaultExternalSrcs(true)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(ngc, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, pols, false).
		Return(expectedGraph)
	expectedResp := &v1.SimulateNetworkGraphResponse{}
	expectedResp.SetSimulatedGraph(expectedGraph)
	expectedResp.SetPolicies([]*v1.NetworkPolicyInSimulation{})

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{}
	request.SetClusterId(fakeClusterID)
	request.SetIncludeNodeDiff(true)
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	protoassert.Equal(suite.T(), expectedResp, actualResp, "response should be output from graph generation")
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithReplacement() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML1}.ToRoxNetworkPolicies()
	pols := []*storage.NetworkPolicy{
		compiledPolicies[0],
	}
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(pols, nil)
	ngc := &storage.NetworkGraphConfig{}
	ngc.SetHideDefaultExternalSrcs(true)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(ngc, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy"), false).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy"), false).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	npm := &storage.NetworkPolicyModification{}
	npm.SetApplyYaml(fakeYAML1)
	request := &v1.SimulateNetworkGraphRequest{}
	request.SetClusterId(fakeClusterID)
	request.SetModification(npm)
	request.SetIncludeNodeDiff(true)
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	protoassert.Equal(suite.T(), expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 1)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_MODIFIED, actualResp.GetPolicies()[0].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithAddition() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML2}.ToRoxNetworkPolicies()
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(compiledPolicies, nil)
	ngc := &storage.NetworkGraphConfig{}
	ngc.SetHideDefaultExternalSrcs(true)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(ngc, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy", "second-policy"), false).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("second-policy"), false).
		Return(expectedGraph)

	npm := &storage.NetworkPolicyModification{}
	npm.SetApplyYaml(fakeYAML1)
	request := &v1.SimulateNetworkGraphRequest{}
	request.SetClusterId(fakeClusterID)
	request.SetModification(npm)
	request.SetIncludeNodeDiff(true)
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	protoassert.Equal(suite.T(), expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 2)
	suite.Equal("second-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_UNCHANGED, actualResp.GetPolicies()[0].GetStatus())
	suite.Equal("first-policy", actualResp.GetPolicies()[1].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[1].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithReplacementAndAddition() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML1}.ToRoxNetworkPolicies()
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(compiledPolicies, nil)
	ngc := &storage.NetworkGraphConfig{}
	ngc.SetHideDefaultExternalSrcs(true)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(ngc, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy", "second-policy"), false).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy"), false).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	npm := &storage.NetworkPolicyModification{}
	npm.SetApplyYaml(combinedYAMLs)
	request := &v1.SimulateNetworkGraphRequest{}
	request.SetClusterId(fakeClusterID)
	request.SetModification(npm)
	request.SetIncludeNodeDiff(true)
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")

	protoassert.Equal(suite.T(), expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 2)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_MODIFIED, actualResp.GetPolicies()[0].GetStatus())
	suite.Equal("second-policy", actualResp.GetPolicies()[1].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[1].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithDeletion() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML1}.ToRoxNetworkPolicies()
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(compiledPolicies, nil)
	ngc := &storage.NetworkGraphConfig{}
	ngc.SetHideDefaultExternalSrcs(true)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(ngc, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies(), false).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy"), false).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	request := v1.SimulateNetworkGraphRequest_builder{
		ClusterId: fakeClusterID,
		Modification: storage.NetworkPolicyModification_builder{
			ToDelete: []*storage.NetworkPolicyReference{
				storage.NetworkPolicyReference_builder{
					Namespace: "default",
					Name:      "first-policy",
				}.Build(),
			},
		}.Build(),
		IncludeNodeDiff: true,
	}.Build()
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")

	protoassert.Equal(suite.T(), expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 1)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetOldPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_DELETED, actualResp.GetPolicies()[0].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithDeletionAndAdditionOfSame() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML2}.ToRoxNetworkPolicies()
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(compiledPolicies, nil)
	ngc := &storage.NetworkGraphConfig{}
	ngc.SetHideDefaultExternalSrcs(true)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(ngc, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy", "second-policy"), false).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("second-policy"), false).
		Return(expectedGraph)

	request := v1.SimulateNetworkGraphRequest_builder{
		ClusterId: fakeClusterID,
		Modification: storage.NetworkPolicyModification_builder{
			ToDelete: []*storage.NetworkPolicyReference{
				storage.NetworkPolicyReference_builder{
					Namespace: "default",
					Name:      "second-policy",
				}.Build(),
			},
			ApplyYaml: combinedYAMLs,
		}.Build(),
		IncludeNodeDiff: true,
	}.Build()
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	protoassert.Equal(suite.T(), expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 2)
	suite.Equal("second-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_MODIFIED, actualResp.GetPolicies()[0].GetStatus())
	suite.Equal("first-policy", actualResp.GetPolicies()[1].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[1].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithOnlyAdditions() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(nil, nil)
	ngc := &storage.NetworkGraphConfig{}
	ngc.SetHideDefaultExternalSrcs(true)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(ngc, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy", "second-policy"), false).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies(), false).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	npm := &storage.NetworkPolicyModification{}
	npm.SetApplyYaml(combinedYAMLs)
	request := &v1.SimulateNetworkGraphRequest{}
	request.SetClusterId(fakeClusterID)
	request.SetModification(npm)
	request.SetIncludeNodeDiff(true)
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	protoassert.Equal(suite.T(), expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 2)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[0].GetStatus())
	suite.Equal("second-policy", actualResp.GetPolicies()[1].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[1].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkPoliciesWithoutDeploymentQuery() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we have network policies in effect for the cluster.
	neps := make([]*storage.NetworkPolicy, 0)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, fakeClusterID, "").
		Return(neps, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.GetNetworkPoliciesRequest{}
	request.SetClusterId(fakeClusterID)
	actualResp, err := suite.tested.GetNetworkPolicies(suite.requestContext, request)

	suite.NoError(err, "expected graph generation to succeed")
	protoassert.SlicesEqual(suite.T(), neps, actualResp.GetNetworkPolicies(), "response should be policies read from store")
}

func (suite *ServiceTestSuite) TestGetNetworkPoliciesWitDeploymentQuery() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we have network policies in effect for the cluster.
	neps := make([]*storage.NetworkPolicy, 0)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, fakeClusterID, "").
		Return(neps, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), testutils.PredMatcher("deployment search is for cluster", func(query *v1.Query) bool {
		// Should be a conjunction with cluster and deployment id.
		conj := query.GetConjunction()
		if len(conj.GetQueries()) != 2 {
			return false
		}
		matchCount := 0
		for _, query := range conj.GetQueries() {
			if queryIsForClusterID(query, fakeClusterID) || queryIsForDeploymentID(query, fakeDeploymentID) {
				matchCount = matchCount + 1
			}
		}
		return matchCount == 2
	})).Return(deps, nil)

	ngc := &storage.NetworkGraphConfig{}
	ngc.SetHideDefaultExternalSrcs(true)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(ngc, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)
	// Check that the evaluator gets called with our created deployment and policy set.
	expectedPolicies := make([]*storage.NetworkPolicy, 0)
	suite.evaluator.EXPECT().GetAppliedPolicies(deps, networkTree, neps).
		Return(expectedPolicies)

	// Make the request to the service and check that it did not err.
	request := &v1.GetNetworkPoliciesRequest{}
	request.SetClusterId(fakeClusterID)
	request.SetDeploymentQuery(fmt.Sprintf("%s:\"%s\"", search.DeploymentID, fakeDeploymentID))
	actualResp, err := suite.tested.GetNetworkPolicies(suite.requestContext, request)

	suite.NoError(err, "expected graph generation to succeed")
	protoassert.SlicesEqual(suite.T(), expectedPolicies, actualResp.GetNetworkPolicies(), "response should be policies applied to deployments")
}

func (suite *ServiceTestSuite) TestGetAllNetworkPoliciesForNamespace() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we have network policies in effect for the cluster.
	neps := make([]*storage.NetworkPolicy, 0)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, fakeClusterID, gomock.Eq("my-namespace")).
		Return(neps, nil).
		Times(1)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, fakeClusterID, gomock.Eq("")).
		Times(0)

	// Make the request to the service and check that it did not err.
	request := &v1.GetNetworkPoliciesRequest{}
	request.SetClusterId(fakeClusterID)
	request.SetNamespace("my-namespace")
	actualResp, err := suite.tested.GetNetworkPolicies(suite.requestContext, request)

	suite.NoError(err, "expected graph generation to succeed")
	protoassert.SlicesEqual(suite.T(), neps, actualResp.GetNetworkPolicies(), "response should be policies read from store")
}

func (suite *ServiceTestSuite) TestGetAllowedPeersFromCurrentPolicyForDeployment() {
	// NOTE: although the test verifies GetAllowedPeersFromCurrentPolicyForDeployment, most of the
	// dependency calls are mocked out. Thus those dependency calls' logics are not tested. This
	// only verifies the needed dependency calls are indeed getting called and also the execution logic
	// of the private functions used by GetAllowedPeersFromCurrentPolicyForDeployment.
	// Prepare deployment001 - deployment004
	numDeployments := 4
	deps := make([]*storage.Deployment, 0, numDeployments)
	for i := 0; i < numDeployments; i++ {
		deployment := &storage.Deployment{}
		deployment.SetId(fmt.Sprintf("deployment%03d", i))
		deployment.SetName(fmt.Sprintf("deployment%03d", i))
		deployment.SetNamespace("namespace")
		deployment.SetClusterId(fakeClusterID)
		deployment.SetPodLabels(map[string]string{"app": fmt.Sprintf("deployment%03d", i)})
		deps = append(deps, deployment)
	}
	suite.deployments.EXPECT().SearchRawDeployments(
		gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).MinTimes(numDeployments).Return(deps, nil)

	var pols []*storage.NetworkPolicy
	suite.evaluator.EXPECT().GetAppliedPolicies(gomock.Any(), gomock.Any(), pols).MinTimes(numDeployments).Return(pols)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").MinTimes(numDeployments).Return(pols, nil)
	ngc := &storage.NetworkGraphConfig{}
	ngc.SetHideDefaultExternalSrcs(true)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(ngc, nil).MinTimes(numDeployments)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).MinTimes(numDeployments).Return(nil)

	// Validate GetAllowedPeers
	for i, testCase := range []struct {
		expectedAllowedPeers []*v1.NetworkBaselineStatusPeer
	}{
		{
			// deployment000
			expectedAllowedPeers: []*v1.NetworkBaselineStatusPeer{
				v1.NetworkBaselineStatusPeer_builder{
					Entity: v1.NetworkBaselinePeerEntity_builder{
						Id:   deps[1].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					}.Build(),
					Port:     80,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					Ingress:  true,
				}.Build(),
				v1.NetworkBaselineStatusPeer_builder{
					Entity: v1.NetworkBaselinePeerEntity_builder{
						Id:   deps[2].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					}.Build(),
					Port:     443,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					Ingress:  false,
				}.Build(),
				v1.NetworkBaselineStatusPeer_builder{
					Entity: v1.NetworkBaselinePeerEntity_builder{
						Id:   deps[2].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					}.Build(),
					Port:     80,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					Ingress:  false,
				}.Build(),
			},
		},
		{
			// deployment001
			expectedAllowedPeers: []*v1.NetworkBaselineStatusPeer{
				v1.NetworkBaselineStatusPeer_builder{
					Entity: v1.NetworkBaselinePeerEntity_builder{
						Id:   deps[0].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					}.Build(),
					Port:     80,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					Ingress:  false,
				}.Build(),
				v1.NetworkBaselineStatusPeer_builder{
					Entity: v1.NetworkBaselinePeerEntity_builder{
						Id:   deps[2].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					}.Build(),
					Port:     0,
					Protocol: storage.L4Protocol_L4_PROTOCOL_ANY,
					Ingress:  true,
				}.Build(),
			},
		},
		{
			// deployment002
			expectedAllowedPeers: []*v1.NetworkBaselineStatusPeer{
				v1.NetworkBaselineStatusPeer_builder{
					Entity: v1.NetworkBaselinePeerEntity_builder{
						Id:   deps[0].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					}.Build(),
					Port:     443,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					Ingress:  true,
				}.Build(),
				v1.NetworkBaselineStatusPeer_builder{
					Entity: v1.NetworkBaselinePeerEntity_builder{
						Id:   deps[0].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					}.Build(),
					Port:     80,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					Ingress:  true,
				}.Build(),
				v1.NetworkBaselineStatusPeer_builder{
					Entity: v1.NetworkBaselinePeerEntity_builder{
						Id:   deps[1].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					}.Build(),
					Port:     0,
					Protocol: storage.L4Protocol_L4_PROTOCOL_ANY,
					Ingress:  false,
				}.Build(),
			},
		},
		{
			// deployment003
			expectedAllowedPeers: nil,
		},
	} {
		suite.Run(fmt.Sprintf("testing deployment%03d", i), func() {
			// Mark testing deployment node's query match to be true
			graph := suite.getSampleNetworkGraph(deps...)
			graph.GetNodes()[i].SetQueryMatch(true)

			suite.evaluator.EXPECT().GetGraph(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(graph)
			suite.deployments.EXPECT().GetDeployment(gomock.Any(), gomock.Any()).Return(deps[i], true, nil)
			rbid := &v1.ResourceByID{}
			rbid.SetId(deps[0].GetId())
			resp, err := suite.tested.GetAllowedPeersFromCurrentPolicyForDeployment(
				suite.requestContext, rbid)
			suite.NoError(err, "expected GetAllowedPeersFromCurrentPolicyForDeployment to succeed")

			protoassert.ElementsMatch(suite.T(), resp.GetAllowedPeers(), testCase.expectedAllowedPeers)
		})
	}
}

func (suite *ServiceTestSuite) TestGetUndoDeploymentRecord() {
	deployment := &storage.Deployment{}
	deployment.SetId("some-deployment")
	deployment.SetNamespace("some-namespace")
	suite.deployments.EXPECT().GetDeployment(gomock.Any(), "some-deployment").Return(
		deployment,
		true,
		nil)
	npaudr := &storage.NetworkPolicyApplicationUndoDeploymentRecord{}
	npaudr.SetDeploymentId("some-deployment")
	npaudr.SetUndoRecord(&storage.NetworkPolicyApplicationUndoRecord{})
	suite.
		networkPolicies.
		EXPECT().
		GetUndoDeploymentRecord(gomock.Any(), "some-deployment").
		Return(
			npaudr,
			true,
			nil)
	rbid := &v1.ResourceByID{}
	rbid.SetId("some-deployment")
	resp, err :=
		suite.tested.GetUndoModificationForDeployment(suite.requestContext, rbid)
	suite.NoError(err)
	gumfdr := &v1.GetUndoModificationForDeploymentResponse{}
	gumfdr.SetUndoRecord(&storage.NetworkPolicyApplicationUndoRecord{})
	protoassert.Equal(suite.T(),
		gumfdr,
		resp)

}

func depToInfo(dep *storage.Deployment) *storage.NetworkEntityInfo {
	nd := &storage.NetworkEntityInfo_Deployment{}
	nd.SetName(dep.GetName())
	nd.SetNamespace(dep.GetNamespace())
	nei := &storage.NetworkEntityInfo{}
	nei.SetType(storage.NetworkEntityInfo_DEPLOYMENT)
	nei.SetId(dep.GetId())
	nei.SetDeployment(proto.ValueOrDefault(nd))
	return nei
}

// getSampleNetworkGraph requires at least 4 deployments
// This function configures a graph which has explicit edges like this:
//   - deployment001 -> deployment000 -> deployment002
//
// deployment003 is an "island" in this graph
// deployment001 has non-isolated ingress, and deployment002 has non-isolated egress. Thus
// there should be an implicit edge from deployment002 -> deployment001
func (suite *ServiceTestSuite) getSampleNetworkGraph(deps ...*storage.Deployment) *v1.NetworkGraph {
	suite.GreaterOrEqual(len(deps), 4)
	return v1.NetworkGraph_builder{
		Epoch: 0,
		Nodes: []*v1.NetworkNode{
			v1.NetworkNode_builder{
				Entity: depToInfo(deps[0]),
				OutEdges: map[int32]*v1.NetworkEdgePropertiesBundle{
					2: v1.NetworkEdgePropertiesBundle_builder{
						Properties: []*v1.NetworkEdgeProperties{
							v1.NetworkEdgeProperties_builder{
								Port:     443,
								Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
							}.Build(),
							v1.NetworkEdgeProperties_builder{
								Port:     80,
								Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
			v1.NetworkNode_builder{
				Entity: depToInfo(deps[1]),
				OutEdges: map[int32]*v1.NetworkEdgePropertiesBundle{
					0: v1.NetworkEdgePropertiesBundle_builder{
						Properties: []*v1.NetworkEdgeProperties{
							v1.NetworkEdgeProperties_builder{
								Port:     80,
								Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
							}.Build(),
						},
					}.Build(),
				},
				NonIsolatedIngress: true,
			}.Build(),
			v1.NetworkNode_builder{
				Entity:            depToInfo(deps[2]),
				NonIsolatedEgress: true,
			}.Build(),
			v1.NetworkNode_builder{
				Entity: depToInfo(deps[3]),
			}.Build(),
		},
	}.Build()
}

// deploymentSearchIsForCluster returns a function that returns true if the in input ParsedSearchRequest has the
// ClusterID field set to the input clusterID.
func deploymentSearchIsForCluster(clusterID string) gomock.Matcher {
	return testutils.PredMatcher("deployment search is for cluster", func(query *v1.Query) bool {
		// Should be a single conjunction with a base string query inside.
		return query.GetBaseQuery().GetMatchFieldQuery().GetValue() == search.ExactMatchString(clusterID)
	})
}

// networkPolicyGetIsForCluster returns a function that returns true if the in input GetNetworkPolicyRequest has the
// ClusterID field set to the input clusterID.
func networkPolicyGetIsForCluster(expectedClusterID string) gomock.Matcher {
	return testutils.PredMatcher("network policy get is for cluster", func(actualClusterID string) bool {
		return actualClusterID == expectedClusterID
	})
}

func queryIsForClusterID(query *v1.Query, clusterID string) bool {
	if query.GetBaseQuery().GetMatchFieldQuery().GetField() != search.ClusterID.String() {
		return false
	}
	return query.GetBaseQuery().GetMatchFieldQuery().GetValue() == search.ExactMatchString(clusterID)
}

func queryIsForDeploymentID(query *v1.Query, deploymentID string) bool {
	if query.GetBaseQuery().GetMatchFieldQuery().GetField() != search.DeploymentID.String() {
		return false
	}
	return query.GetBaseQuery().GetMatchFieldQuery().GetValue() == search.ExactMatchString(deploymentID)
}

// checkHasPolicies returns a function that returns true if the input is a slice of network policies, containing
// exactly one policy for every input (policyNames).
func checkHasPolicies(policyNames ...string) gomock.Matcher {
	return testutils.PredMatcher("has policies", func(networkPolicies []*storage.NetworkPolicy) bool {
		if len(networkPolicies) != len(policyNames) {
			return false
		}
		for _, name := range policyNames {
			count := 0
			for _, policy := range networkPolicies {
				if policy.GetName() == name {
					count = count + 1
				}
			}
			if count != 1 {
				return false
			}
		}
		return true
	})
}

func TestCheckAllNamespacesWriteAllowed(t *testing.T) {

	namespaces := []string{"foo", "bar", "baz", "qux"}
	clusterID := "clusterA"

	cases := map[string]struct {
		checker       sac.ScopeCheckerCore
		expectAllowed bool
	}{
		"all access": {
			checker:       sac.AllowAllAccessScopeChecker(),
			expectAllowed: true,
		},
		"access to clusterA": {
			checker: sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.NetworkPolicy),
				sac.ClusterScopeKeys("clusterA")),
			expectAllowed: true,
		},
		"exactly matching namespaces": {
			checker: sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.NetworkPolicy),
				sac.ClusterScopeKeys("clusterA"),
				sac.NamespaceScopeKeys("foo", "bar", "baz", "qux")),
			expectAllowed: true,
		},
		"more namespaces": {
			checker: sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.NetworkPolicy),
				sac.ClusterScopeKeys("clusterA"),
				sac.NamespaceScopeKeys("foo", "bar", "baz", "qux", "quuz")),
			expectAllowed: true,
		},
		"no access": {
			checker:       sac.DenyAllAccessScopeChecker(),
			expectAllowed: false,
		},
		"access to clusterB": {
			checker: sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.NetworkPolicy),
				sac.ClusterScopeKeys("clusterB")),
			expectAllowed: false,
		},
		"correct namespaces in wrong cluster": {
			checker: sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.NetworkPolicy),
				sac.ClusterScopeKeys("clusterB"),
				sac.NamespaceScopeKeys("foo", "bar", "baz", "qux")),
			expectAllowed: false,
		},
		"one namespace missing": {
			checker: sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.NetworkPolicy),
				sac.ClusterScopeKeys("clusterA"),
				sac.NamespaceScopeKeys("bar", "baz", "qux")),
			expectAllowed: false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := sac.WithGlobalAccessScopeChecker(context.Background(), c.checker)
			err := checkAllNamespacesWriteAllowed(ctx, clusterID, namespaces...)
			if c.expectAllowed {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, sac.ErrResourceAccessDenied)
			}
		})
	}
}

func TestGetNamespacesFromModification(t *testing.T) {

	cases := map[string]struct {
		applyYAML string
		toDelete  []*storage.NetworkPolicyReference

		expectedNamespaces []string
	}{
		"single policy in default ns": {
			applyYAML:          fakeYAML1,
			expectedNamespaces: []string{"default"},
		},
		"single deletion in test ns": {
			toDelete: []*storage.NetworkPolicyReference{
				storage.NetworkPolicyReference_builder{
					Name:      "foo",
					Namespace: "testns",
				}.Build(),
			},
			expectedNamespaces: []string{"testns"},
		},
		"multi-document YAML and deletion": {
			applyYAML: combinedYAMLs,
			toDelete: []*storage.NetworkPolicyReference{
				storage.NetworkPolicyReference_builder{
					Name:      "foo",
					Namespace: "testns",
				}.Build(),
			},
			expectedNamespaces: []string{"default", "testns"},
		},
		"yaml with empty namespace": {
			applyYAML:          badYAML,
			expectedNamespaces: []string{""},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {

			mod := &storage.NetworkPolicyModification{}
			mod.SetApplyYaml(c.applyYAML)
			mod.SetToDelete(c.toDelete)

			nsSet, err := getNamespacesFromModification(mod)
			require.NoError(t, err)
			assert.ElementsMatch(t, nsSet.AsSlice(), c.expectedNamespaces)
		})
	}
}
