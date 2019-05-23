package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	cDataStoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	dDataStoreMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	npMocks "github.com/stackrox/rox/central/networkpolicies/datastore/mocks"
	npGraphMocks "github.com/stackrox/rox/central/networkpolicies/graph/mocks"
	nDataStoreMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	grpcTestutils "github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
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

	requestContext  context.Context
	clusters        *cDataStoreMocks.MockDataStore
	deployments     *dDataStoreMocks.MockDataStore
	networkPolicies *npMocks.MockDataStore
	evaluator       *npGraphMocks.MockEvaluator
	notifiers       *nDataStoreMocks.MockDataStore
	tested          Service

	mockCtrl *gomock.Controller
}

func (suite *ServiceTestSuite) SetupTest() {
	// Since all the datastores underneath are mocked, the context of the request doesns't need any permissions.
	suite.requestContext = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.networkPolicies = npMocks.NewMockDataStore(suite.mockCtrl)
	suite.evaluator = npGraphMocks.NewMockEvaluator(suite.mockCtrl)
	suite.clusters = cDataStoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.deployments = dDataStoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.notifiers = nDataStoreMocks.NewMockDataStore(suite.mockCtrl)

	suite.tested = New(suite.networkPolicies, suite.deployments, suite.evaluator, nil, suite.clusters, suite.notifiers, nil, nil)
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
	// Mock that cluster exists.
	suite.clusters.EXPECT().GetCluster(gomock.Any(), fakeClusterID).
		Return((*storage.Cluster)(nil), false, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
	}
	_, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.Error(err, "expected graph generation to fail since cluster does not exist")
}

func (suite *ServiceTestSuite) TestRejectsYamlWithoutNamespace() {
	// Mock that cluster exists.
	cluster := &storage.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(gomock.Any(), fakeClusterID).
		Return(cluster, true, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
		Modification: &storage.NetworkPolicyModification{
			ApplyYaml: badYAML,
		},
	}
	_, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.Error(err, "expected graph generation to fail since input yaml has no namespace")
}

func (suite *ServiceTestSuite) TestGetNetworkGraph() {
	// Mock that cluster exists.
	cluster := &storage.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(gomock.Any(), fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	pols := make([]*storage.NetworkPolicy, 0)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(pols, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(deps, pols).
		Return(expectedGraph)
	expectedResp := &v1.SimulateNetworkGraphResponse{
		SimulatedGraph: expectedGraph,
		Policies:       []*v1.NetworkPolicyInSimulation{},
	}

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
	}
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedResp, actualResp, "response should be output from graph generation")
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithReplacement() {
	// Mock that cluster exists.
	cluster := &storage.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(gomock.Any(), fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML1}.ToRoxNetworkPolicies()
	pols := []*storage.NetworkPolicy{
		compiledPolicies[0],
	}
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(pols, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies("first-policy")).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies("first-policy")).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
		Modification: &storage.NetworkPolicyModification{
			ApplyYaml: fakeYAML1,
		},
	}
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 1)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_MODIFIED, actualResp.GetPolicies()[0].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithAddition() {
	// Mock that cluster exists.
	cluster := &storage.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(gomock.Any(), fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML2}.ToRoxNetworkPolicies()
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(compiledPolicies, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies("first-policy", "second-policy")).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies("second-policy")).
		Return(expectedGraph)

	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
		Modification: &storage.NetworkPolicyModification{
			ApplyYaml: fakeYAML1,
		},
	}
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 2)
	suite.Equal("second-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_UNCHANGED, actualResp.GetPolicies()[0].GetStatus())
	suite.Equal("first-policy", actualResp.GetPolicies()[1].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[1].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithReplacementAndAddition() {
	// Mock that cluster exists.
	cluster := &storage.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(gomock.Any(), fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML1}.ToRoxNetworkPolicies()
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(compiledPolicies, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies("first-policy", "second-policy")).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies("first-policy")).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
		Modification: &storage.NetworkPolicyModification{
			ApplyYaml: combinedYAMLs,
		},
	}
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")

	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 2)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_MODIFIED, actualResp.GetPolicies()[0].GetStatus())
	suite.Equal("second-policy", actualResp.GetPolicies()[1].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[1].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithDeletion() {
	// Mock that cluster exists.
	cluster := &storage.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(gomock.Any(), fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML1}.ToRoxNetworkPolicies()
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(compiledPolicies, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies()).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies("first-policy")).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
		Modification: &storage.NetworkPolicyModification{
			ToDelete: []*storage.NetworkPolicyReference{
				{
					Namespace: "default",
					Name:      "first-policy",
				},
			},
		},
	}
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")

	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 1)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetOldPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_DELETED, actualResp.GetPolicies()[0].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithDeletionAndAdditionOfSame() {
	// Mock that cluster exists.
	cluster := &storage.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(gomock.Any(), fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML2}.ToRoxNetworkPolicies()
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(compiledPolicies, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies("first-policy", "second-policy")).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies("second-policy")).
		Return(expectedGraph)

	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
		Modification: &storage.NetworkPolicyModification{
			ToDelete: []*storage.NetworkPolicyReference{
				{
					Namespace: "default",
					Name:      "second-policy",
				},
			},
			ApplyYaml: combinedYAMLs,
		},
	}
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 2)
	suite.Equal("second-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_MODIFIED, actualResp.GetPolicies()[0].GetStatus())
	suite.Equal("first-policy", actualResp.GetPolicies()[1].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[1].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithOnlyAdditions() {
	// Mock that cluster exists.
	cluster := &storage.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(gomock.Any(), fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(nil, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies("first-policy", "second-policy")).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies()).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
		Modification: &storage.NetworkPolicyModification{
			ApplyYaml: combinedYAMLs,
		},
	}
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 2)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[0].GetStatus())
	suite.Equal("second-policy", actualResp.GetPolicies()[1].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[1].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkPoliciesWithoutDeploymentQuery() {
	// Mock that cluster exists.
	cluster := &storage.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(gomock.Any(), fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we have network policies in effect for the cluster.
	neps := make([]*storage.NetworkPolicy, 0)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, fakeClusterID, "").
		Return(neps, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.GetNetworkPoliciesRequest{
		ClusterId: fakeClusterID,
	}
	actualResp, err := suite.tested.GetNetworkPolicies(suite.requestContext, request)

	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(neps, actualResp.GetNetworkPolicies(), "response should be policies read from store")
}

func (suite *ServiceTestSuite) TestGetNetworkPoliciesWitDeploymentQuery() {
	// Mock that cluster exists.
	cluster := &storage.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(gomock.Any(), fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we have network policies in effect for the cluster.
	neps := make([]*storage.NetworkPolicy, 0)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, fakeClusterID, "").
		Return(neps, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
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

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedPolicies := make([]*storage.NetworkPolicy, 0)
	suite.evaluator.EXPECT().GetAppliedPolicies(deps, neps).
		Return(expectedPolicies)

	// Make the request to the service and check that it did not err.
	request := &v1.GetNetworkPoliciesRequest{
		ClusterId:       fakeClusterID,
		DeploymentQuery: fmt.Sprintf("%s:\"%s\"", search.DeploymentID, fakeDeploymentID),
	}
	actualResp, err := suite.tested.GetNetworkPolicies(suite.requestContext, request)

	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedPolicies, actualResp.GetNetworkPolicies(), "response should be policies applied to deployments")
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
