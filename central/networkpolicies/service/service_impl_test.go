package service

import (
	"context"
	"testing"

	cDataStoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	dDataStoreMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	ngMocks "github.com/stackrox/rox/central/networkgraph/mocks"
	npStoreMocks "github.com/stackrox/rox/central/networkpolicies/store/mocks"
	notifierStoreMocks "github.com/stackrox/rox/central/notifier/store/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stretchr/testify/mock"
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

	clusters        *cDataStoreMocks.DataStore
	deployments     *dDataStoreMocks.DataStore
	networkPolicies *npStoreMocks.Store
	evaluator       *ngMocks.Evaluator
	notifiers       *notifierStoreMocks.Store
	tested          Service
}

func (suite *ServiceTestSuite) SetupTest() {
	suite.networkPolicies = &npStoreMocks.Store{}
	suite.evaluator = &ngMocks.Evaluator{}
	suite.clusters = &cDataStoreMocks.DataStore{}
	suite.deployments = &dDataStoreMocks.DataStore{}
	suite.notifiers = &notifierStoreMocks.Store{}

	suite.tested = New(suite.networkPolicies, suite.deployments, suite.evaluator, suite.clusters, suite.notifiers)
}

func (suite *ServiceTestSuite) TestFailsIfClusterIsNotSet() {
	request := &v1.GetNetworkGraphRequest{}
	_, err := suite.tested.GetNetworkGraph((context.Context)(nil), request)
	suite.Error(err, "expected graph generation to fail since no cluster is specified")

	suite.assertAllExpectationsMet()
}

func (suite *ServiceTestSuite) TestFailsIfClusterDoesNotExist() {
	// Mock that cluster exists.
	suite.clusters.On("GetCluster", fakeClusterID).
		Return((*v1.Cluster)(nil), false, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.GetNetworkGraphRequest{
		ClusterId: fakeClusterID,
	}
	_, err := suite.tested.GetNetworkGraph((context.Context)(nil), request)
	suite.Error(err, "expected graph generation to fail since cluster does not exist")

	suite.assertAllExpectationsMet()
}

func (suite *ServiceTestSuite) TestRejectsYamlWithoutNamespace() {
	// Mock that cluster exists.
	cluster := &v1.Cluster{Id: fakeClusterID}
	suite.clusters.On("GetCluster", fakeClusterID).
		Return(cluster, true, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.GetNetworkGraphRequest{
		ClusterId:      fakeClusterID,
		SimulationYaml: badYAML,
	}
	_, err := suite.tested.GetNetworkGraph((context.Context)(nil), request)
	suite.Error(err, "expected graph generation to fail since input yaml has no namespace")

	suite.assertAllExpectationsMet()
}

func (suite *ServiceTestSuite) TestGetNetworkGraph() {
	// Mock that cluster exists.
	cluster := &v1.Cluster{Id: fakeClusterID}
	suite.clusters.On("GetCluster", fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*v1.Deployment, 0)
	suite.deployments.On("SearchRawDeployments", mock.MatchedBy(deploymentSearchIsForCluster(fakeClusterID))).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	pols := make([]*v1.NetworkPolicy, 0)
	suite.networkPolicies.On("GetNetworkPolicies", mock.MatchedBy(networkPolicyGetIsForCluster(fakeClusterID))).
		Return(pols, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedResp := &v1.GetNetworkGraphResponse{}
	suite.evaluator.On("GetGraph", deps, pols).
		Return(expectedResp, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.GetNetworkGraphRequest{
		ClusterId: fakeClusterID,
	}
	actualResp, err := suite.tested.GetNetworkGraph((context.Context)(nil), request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedResp, actualResp, "response should be output from graph generation")

	suite.assertAllExpectationsMet()
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithReplacement() {
	// Mock that cluster exists.
	cluster := &v1.Cluster{Id: fakeClusterID}
	suite.clusters.On("GetCluster", fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*v1.Deployment, 0)
	suite.deployments.On("SearchRawDeployments", mock.MatchedBy(deploymentSearchIsForCluster(fakeClusterID))).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML1}.ToRoxNetworkPolicies()
	pols := []*v1.NetworkPolicy{
		compiledPolicies[0],
	}
	suite.networkPolicies.On("GetNetworkPolicies", mock.MatchedBy(networkPolicyGetIsForCluster(fakeClusterID))).
		Return(pols, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedResp := &v1.GetNetworkGraphResponse{}
	suite.evaluator.On("GetGraph", deps, mock.MatchedBy(checkHasPolicies("first-policy"))).
		Return(expectedResp, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.GetNetworkGraphRequest{
		ClusterId:      fakeClusterID,
		SimulationYaml: fakeYAML1,
	}
	actualResp, err := suite.tested.GetNetworkGraph((context.Context)(nil), request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedResp, actualResp, "response should be output from graph generation")

	suite.assertAllExpectationsMet()
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithAddition() {
	// Mock that cluster exists.
	cluster := &v1.Cluster{Id: fakeClusterID}
	suite.clusters.On("GetCluster", fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*v1.Deployment, 0)
	suite.deployments.On("SearchRawDeployments", mock.MatchedBy(deploymentSearchIsForCluster(fakeClusterID))).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML2}.ToRoxNetworkPolicies()
	suite.networkPolicies.On("GetNetworkPolicies", mock.MatchedBy(networkPolicyGetIsForCluster(fakeClusterID))).
		Return(compiledPolicies, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedResp := &v1.GetNetworkGraphResponse{}
	suite.evaluator.On("GetGraph", deps, mock.MatchedBy(checkHasPolicies("first-policy", "second-policy"))).
		Return(expectedResp, nil)

	request := &v1.GetNetworkGraphRequest{
		ClusterId:      fakeClusterID,
		SimulationYaml: fakeYAML1,
	}
	actualResp, err := suite.tested.GetNetworkGraph((context.Context)(nil), request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedResp, actualResp, "response should be output from graph generation")

	suite.assertAllExpectationsMet()
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithReplacementAndAddition() {
	// Mock that cluster exists.
	cluster := &v1.Cluster{Id: fakeClusterID}
	suite.clusters.On("GetCluster", fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*v1.Deployment, 0)
	suite.deployments.On("SearchRawDeployments", mock.MatchedBy(deploymentSearchIsForCluster(fakeClusterID))).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML1}.ToRoxNetworkPolicies()
	suite.networkPolicies.On("GetNetworkPolicies", mock.MatchedBy(networkPolicyGetIsForCluster(fakeClusterID))).
		Return(compiledPolicies, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedResp := &v1.GetNetworkGraphResponse{}
	suite.evaluator.On("GetGraph", deps, mock.MatchedBy(checkHasPolicies("first-policy", "second-policy"))).
		Return(expectedResp, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.GetNetworkGraphRequest{
		ClusterId:      fakeClusterID,
		SimulationYaml: combinedYAMLs,
	}
	actualResp, err := suite.tested.GetNetworkGraph((context.Context)(nil), request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedResp, actualResp, "response should be output from graph generation")

	suite.assertAllExpectationsMet()
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithOnlyAdditions() {
	// Mock that cluster exists.
	cluster := &v1.Cluster{Id: fakeClusterID}
	suite.clusters.On("GetCluster", fakeClusterID).
		Return(cluster, true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*v1.Deployment, 0)
	suite.deployments.On("SearchRawDeployments", mock.MatchedBy(deploymentSearchIsForCluster(fakeClusterID))).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	suite.networkPolicies.On("GetNetworkPolicies", mock.MatchedBy(networkPolicyGetIsForCluster(fakeClusterID))).
		Return(nil, nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedResp := &v1.GetNetworkGraphResponse{}
	suite.evaluator.On("GetGraph", deps, mock.MatchedBy(checkHasPolicies("first-policy", "second-policy"))).
		Return(expectedResp, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.GetNetworkGraphRequest{
		ClusterId:      fakeClusterID,
		SimulationYaml: combinedYAMLs,
	}
	actualResp, err := suite.tested.GetNetworkGraph((context.Context)(nil), request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedResp, actualResp, "response should be output from graph generation")

	suite.assertAllExpectationsMet()
}

func (suite *ServiceTestSuite) assertAllExpectationsMet() {
	suite.networkPolicies.AssertExpectations(suite.T())
	suite.evaluator.AssertExpectations(suite.T())
	suite.clusters.AssertExpectations(suite.T())
	suite.deployments.AssertExpectations(suite.T())
}

// deploymentSearchIsForCluster returns a function that returns true if the in input ParsedSearchRequest has the
// ClusterID field set to the input clusterID.
func deploymentSearchIsForCluster(clusterID string) func(in interface{}) bool {
	return func(in interface{}) bool {
		query, isQuery := in.(*v1.Query)
		if !isQuery {
			return false
		}
		// Should be a single conjunction with a base string query inside.
		return query.GetBaseQuery().GetMatchFieldQuery().GetValue() == clusterID
	}
}

// networkPolicyGetIsForCluster returns a function that returns true if the in input GetNetworkPolicyRequest has the
// ClusterID field set to the input clusterID.
func networkPolicyGetIsForCluster(clusterID string) func(in interface{}) bool {
	return func(in interface{}) bool {
		request, isNPRequest := in.(*v1.GetNetworkPoliciesRequest)
		if !isNPRequest {
			return false
		}
		return request.ClusterId == clusterID
	}
}

// checkHasPolicies returns a function that returns true if the input is a slice of network policies, containing
// exactly one policy for every input (policyNames).
func checkHasPolicies(policyNames ...string) func(in interface{}) bool {
	return func(in interface{}) bool {
		networkPolicies, isNetworkPolicySlice := in.([]*v1.NetworkPolicy)
		if !isNetworkPolicySlice {
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
	}
}
