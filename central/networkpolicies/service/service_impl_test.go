package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	cDataStoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	dDataStoreMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	npGraphMocks "github.com/stackrox/rox/central/networkpolicies/graph/mocks"
	npStoreMocks "github.com/stackrox/rox/central/networkpolicies/store/mocks"
	notifierStoreMocks "github.com/stackrox/rox/central/notifier/store/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

const fakeClusterID = "FAKECLUSTERID"
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

	clusters        *cDataStoreMocks.MockDataStore
	deployments     *dDataStoreMocks.MockDataStore
	networkPolicies *npStoreMocks.MockStore
	evaluator       *npGraphMocks.MockEvaluator
	notifiers       *notifierStoreMocks.MockStore
	tested          Service

	mockCtrl *gomock.Controller
}

func (suite *ServiceTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.networkPolicies = npStoreMocks.NewMockStore(suite.mockCtrl)
	suite.evaluator = npGraphMocks.NewMockEvaluator(suite.mockCtrl)
	suite.clusters = cDataStoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.deployments = dDataStoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.notifiers = notifierStoreMocks.NewMockStore(suite.mockCtrl)

	suite.tested = New(suite.networkPolicies, suite.deployments, suite.evaluator, suite.clusters, suite.notifiers)
}

func (suite *ServiceTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *ServiceTestSuite) TestFailsIfClusterIsNotSet() {
	request := &v1.SimulateNetworkGraphRequest{}
	_, err := suite.tested.SimulateNetworkGraph((context.Context)(nil), request)
	suite.Error(err, "expected graph generation to fail since no cluster is specified")
}

func (suite *ServiceTestSuite) TestFailsIfClusterDoesNotExist() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().GetCluster(fakeClusterID).
		Return((*v1.Cluster)(nil), false, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
	}
	_, err := suite.tested.SimulateNetworkGraph((context.Context)(nil), request)
	suite.Error(err, "expected graph generation to fail since cluster does not exist")
}

func (suite *ServiceTestSuite) TestRejectsYamlWithoutNamespace() {
	// Mock that cluster exists.
	cluster := &v1.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(fakeClusterID).
		Return(cluster, true, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId:      fakeClusterID,
		SimulationYaml: badYAML,
	}
	_, err := suite.tested.SimulateNetworkGraph((context.Context)(nil), request)
	suite.Error(err, "expected graph generation to fail since input yaml has no namespace")
}

func (suite *ServiceTestSuite) TestGetNetworkGraph() {
	// Mock that cluster exists.
	cluster := &v1.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*v1.Deployment, 0)
	suite.deployments.EXPECT().SearchRawDeployments(deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	pols := make([]*v1.NetworkPolicy, 0)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(networkPolicyGetIsForCluster(fakeClusterID)).
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
	actualResp, err := suite.tested.SimulateNetworkGraph((context.Context)(nil), request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedResp, actualResp, "response should be output from graph generation")
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithReplacement() {
	// Mock that cluster exists.
	cluster := &v1.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*v1.Deployment, 0)
	suite.deployments.EXPECT().SearchRawDeployments(deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML1}.ToRoxNetworkPolicies()
	pols := []*v1.NetworkPolicy{
		compiledPolicies[0],
	}
	suite.networkPolicies.EXPECT().GetNetworkPolicies(networkPolicyGetIsForCluster(fakeClusterID)).
		Return(pols, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies("first-policy")).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId:      fakeClusterID,
		SimulationYaml: fakeYAML1,
	}
	actualResp, err := suite.tested.SimulateNetworkGraph((context.Context)(nil), request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 1)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_MODIFIED, actualResp.GetPolicies()[0].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithAddition() {
	// Mock that cluster exists.
	cluster := &v1.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*v1.Deployment, 0)
	suite.deployments.EXPECT().SearchRawDeployments(deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML2}.ToRoxNetworkPolicies()
	suite.networkPolicies.EXPECT().GetNetworkPolicies(networkPolicyGetIsForCluster(fakeClusterID)).
		Return(compiledPolicies, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies("first-policy", "second-policy")).
		Return(expectedGraph)

	request := &v1.SimulateNetworkGraphRequest{
		ClusterId:      fakeClusterID,
		SimulationYaml: fakeYAML1,
	}
	actualResp, err := suite.tested.SimulateNetworkGraph((context.Context)(nil), request)
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
	cluster := &v1.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*v1.Deployment, 0)
	suite.deployments.EXPECT().SearchRawDeployments(deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML1}.ToRoxNetworkPolicies()
	suite.networkPolicies.EXPECT().GetNetworkPolicies(networkPolicyGetIsForCluster(fakeClusterID)).
		Return(compiledPolicies, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies("first-policy", "second-policy")).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId:      fakeClusterID,
		SimulationYaml: combinedYAMLs,
	}
	actualResp, err := suite.tested.SimulateNetworkGraph((context.Context)(nil), request)
	suite.NoError(err, "expected graph generation to succeed")

	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 2)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_MODIFIED, actualResp.GetPolicies()[0].GetStatus())
	suite.Equal("second-policy", actualResp.GetPolicies()[1].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[1].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithOnlyAdditions() {
	// Mock that cluster exists.
	cluster := &v1.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().GetCluster(fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*v1.Deployment, 0)
	suite.deployments.EXPECT().SearchRawDeployments(deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	suite.networkPolicies.EXPECT().GetNetworkPolicies(networkPolicyGetIsForCluster(fakeClusterID)).
		Return(nil, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(deps, checkHasPolicies("first-policy", "second-policy")).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId:      fakeClusterID,
		SimulationYaml: combinedYAMLs,
	}
	actualResp, err := suite.tested.SimulateNetworkGraph((context.Context)(nil), request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 2)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[0].GetStatus())
	suite.Equal("second-policy", actualResp.GetPolicies()[1].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[1].GetStatus())
}

// deploymentSearchIsForCluster returns a function that returns true if the in input ParsedSearchRequest has the
// ClusterID field set to the input clusterID.
func deploymentSearchIsForCluster(clusterID string) gomock.Matcher {
	return testutils.PredMatcher("deployment search is for cluster", func(query *v1.Query) bool {
		// Should be a single conjunction with a base string query inside.
		return query.GetBaseQuery().GetMatchFieldQuery().GetValue() == "="+clusterID
	})
}

// networkPolicyGetIsForCluster returns a function that returns true if the in input GetNetworkPolicyRequest has the
// ClusterID field set to the input clusterID.
func networkPolicyGetIsForCluster(clusterID string) gomock.Matcher {
	return testutils.PredMatcher("network policy get is for cluster", func(request *v1.GetNetworkPoliciesRequest) bool {
		return request.ClusterId == clusterID
	})
}

// checkHasPolicies returns a function that returns true if the input is a slice of network policies, containing
// exactly one policy for every input (policyNames).
func checkHasPolicies(policyNames ...string) gomock.Matcher {
	return testutils.PredMatcher("has policies", func(networkPolicies []*v1.NetworkPolicy) bool {
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
