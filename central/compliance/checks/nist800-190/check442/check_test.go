package check442

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestCheck(t *testing.T) {
	suite.Run(t, new(suiteImpl))
}

type suiteImpl struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *suiteImpl) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *suiteImpl) TearDownSuite() {
	s.mockCtrl.Finish()
}

func (s *suiteImpl) TestHostNetwork() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	deployment := &storage.Deployment{}
	deployment.SetId(uuid.NewV4().String())
	deployment.SetHostNetwork(true)
	testDeployments := []*storage.Deployment{
		deployment,
	}

	testNodes := s.nodes()

	testPolicies := s.networkPolicies()

	testDeploymentsToNetworkPolicies := map[string][]*storage.NetworkPolicy{
		testDeployments[0].GetId(): {testPolicies[0], testPolicies[1]},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().DeploymentsToNetworkPolicies().AnyTimes().Return(testDeploymentsToNetworkPolicies)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	s.NotNil(checkResults)

	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		s.NoError(deploymentResults.Error())
		s.Len(deploymentResults.Evidence(), 1)
		s.Equal(framework.FailStatus, deploymentResults.Evidence()[0].Status)
	}
}

func (s *suiteImpl) TestMissingIngress() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	deployment := &storage.Deployment{}
	deployment.SetId(uuid.NewV4().String())
	deployment.SetHostNetwork(false)
	testDeployments := []*storage.Deployment{
		deployment,
	}

	testNodes := s.nodes()

	testPolicies := s.networkPolicies()

	testDeploymentsToNetworkPolicies := map[string][]*storage.NetworkPolicy{
		testDeployments[0].GetId(): {testPolicies[1]},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().DeploymentsToNetworkPolicies().AnyTimes().Return(testDeploymentsToNetworkPolicies)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	s.NotNil(checkResults)

	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		s.NoError(deploymentResults.Error())
		s.Len(deploymentResults.Evidence(), 1)
		s.Equal(framework.FailStatus, deploymentResults.Evidence()[0].Status)
	}
}

func (s *suiteImpl) TestMissingEgress() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	deployment := &storage.Deployment{}
	deployment.SetId(uuid.NewV4().String())
	deployment.SetHostNetwork(false)
	testDeployments := []*storage.Deployment{
		deployment,
	}

	testNodes := s.nodes()

	testPolicies := s.networkPolicies()

	testDeploymentsToNetworkPolicies := map[string][]*storage.NetworkPolicy{
		testDeployments[0].GetId(): {testPolicies[0]},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().DeploymentsToNetworkPolicies().AnyTimes().Return(testDeploymentsToNetworkPolicies)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	s.NotNil(checkResults)

	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		s.NoError(deploymentResults.Error())
		s.Len(deploymentResults.Evidence(), 1)
		s.Equal(framework.FailStatus, deploymentResults.Evidence()[0].Status)
	}
}

func (s *suiteImpl) TestSkipKubeSystem() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	deployment := &storage.Deployment{}
	deployment.SetId(uuid.NewV4().String())
	deployment.SetHostNetwork(true)
	deployment.SetNamespace("kube-system")
	testDeployments := []*storage.Deployment{
		deployment,
	}

	testNodes := s.nodes()
	testDeploymentsToNetworkPolicies := map[string][]*storage.NetworkPolicy{}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().DeploymentsToNetworkPolicies().AnyTimes().Return(testDeploymentsToNetworkPolicies)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	s.NotNil(checkResults)

	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		s.NoError(deploymentResults.Error())
		if s.Len(deploymentResults.Evidence(), 1) {
			s.Equal(framework.SkipStatus, deploymentResults.Evidence()[0].Status)
		}
	}
}

func (s *suiteImpl) TestPass() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	deployment := &storage.Deployment{}
	deployment.SetId(uuid.NewV4().String())
	deployment.SetHostNetwork(false)
	deployment2 := &storage.Deployment{}
	deployment2.SetId(uuid.NewV4().String())
	deployment2.SetHostNetwork(false)
	testDeployments := []*storage.Deployment{
		deployment,
		deployment2,
	}

	testNodes := s.nodes()

	testPolicies := s.networkPolicies()

	testDeploymentsToNetworkPolicies := map[string][]*storage.NetworkPolicy{
		testDeployments[0].GetId(): {testPolicies[0], testPolicies[1]},
		testDeployments[1].GetId(): {testPolicies[0], testPolicies[1]},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().DeploymentsToNetworkPolicies().AnyTimes().Return(testDeploymentsToNetworkPolicies)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	s.NotNil(checkResults)

	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		s.NoError(deploymentResults.Error())
		s.Len(deploymentResults.Evidence(), 1)
		s.Equal(framework.PassStatus, deploymentResults.Evidence()[0].Status)
	}
}

// Helper functions for test data.
//////////////////////////////////

func (s *suiteImpl) verifyCheckRegistered() framework.Check {
	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	s.NotNil(check)
	return check
}

func (s *suiteImpl) cluster() *storage.Cluster {
	cluster := &storage.Cluster{}
	cluster.SetId(uuid.NewV4().String())
	return cluster
}

func (s *suiteImpl) networkPolicies() []*storage.NetworkPolicy {
	return []*storage.NetworkPolicy{
		storage.NetworkPolicy_builder{
			Id: uuid.NewV4().String(),
			Spec: storage.NetworkPolicySpec_builder{
				PolicyTypes: []storage.NetworkPolicyType{
					storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE,
				},
				Ingress: []*storage.NetworkPolicyIngressRule{
					{},
				},
			}.Build(),
		}.Build(),
		storage.NetworkPolicy_builder{
			Id: uuid.NewV4().String(),
			Spec: storage.NetworkPolicySpec_builder{
				PolicyTypes: []storage.NetworkPolicyType{
					storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE,
				},
				Egress: []*storage.NetworkPolicyEgressRule{
					{},
				},
			}.Build(),
		}.Build(),
	}
}

func (s *suiteImpl) nodes() []*storage.Node {
	node := &storage.Node{}
	node.SetId(uuid.NewV4().String())
	node2 := &storage.Node{}
	node2.SetId(uuid.NewV4().String())
	return []*storage.Node{
		node,
		node2,
	}
}
