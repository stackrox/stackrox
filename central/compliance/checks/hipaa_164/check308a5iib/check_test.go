package check308a5iib

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkentity"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestCheck(t *testing.T) {
	t.Parallel()
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

func (s *suiteImpl) TestFail() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()
	testNodes := s.nodes()
	testDeployments := []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
		},
	}

	testNetworkPolicies := s.networkPolicies()
	testNetworkGraph := &v1.NetworkGraph{
		Nodes: []*v1.NetworkNode{
			{
				Entity:    networkentity.ForDeployment(testDeployments[0].GetId()).ToProto(),
				PolicyIds: []string{testNetworkPolicies[0].GetId()},
			},
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().NetworkPolicies().AnyTimes().Return(toMap(testNetworkPolicies))
	data.EXPECT().NetworkGraph().AnyTimes().Return(testNetworkGraph)
	data.EXPECT().Policies().AnyTimes().Return(nil)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments)
	err = run.Run(context.Background(), domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[checkID]
	s.NotNil(checkResults)

	s.NoError(checkResults.Error())
	s.Len(checkResults.Evidence(), 3)
	s.Equal(framework.FailStatus, checkResults.Evidence()[0].Status)
	s.Equal(framework.FailStatus, checkResults.Evidence()[1].Status)
	s.Equal(framework.FailStatus, checkResults.Evidence()[2].Status)

	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		s.NoError(deploymentResults.Error())
		s.Len(deploymentResults.Evidence(), 1)
		s.Equal(framework.FailStatus, deploymentResults.Evidence()[0].Status)
	}
}

func (s *suiteImpl) TestPass() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	testDeployments := []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
		},
	}

	testNodes := s.nodes()

	testNetworkPolicies := s.networkPolicies()
	testPolicies := s.policies()

	testNetworkGraph := &v1.NetworkGraph{
		Nodes: []*v1.NetworkNode{
			{
				Entity:    networkentity.ForDeployment(testDeployments[0].GetId()).ToProto(),
				PolicyIds: []string{testNetworkPolicies[0].GetId(), testNetworkPolicies[1].GetId()},
			},
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().NetworkPolicies().AnyTimes().Return(toMap(testNetworkPolicies))
	data.EXPECT().NetworkGraph().AnyTimes().Return(testNetworkGraph)
	data.EXPECT().Policies().AnyTimes().Return(testPolicies)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments)
	err = run.Run(context.Background(), domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[checkID]
	s.NotNil(checkResults)

	s.NoError(checkResults.Error())
	s.Len(checkResults.Evidence(), 3)
	s.Equal(framework.PassStatus, checkResults.Evidence()[0].Status)
	s.Equal(framework.PassStatus, checkResults.Evidence()[1].Status)
	s.Equal(framework.PassStatus, checkResults.Evidence()[2].Status)

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
	check := registry.Lookup(checkID)
	s.NotNil(check)
	return check
}

func (s *suiteImpl) cluster() *storage.Cluster {
	return &storage.Cluster{
		Id: uuid.NewV4().String(),
	}
}

func (s *suiteImpl) networkPolicies() []*storage.NetworkPolicy {
	return []*storage.NetworkPolicy{
		{
			Id: uuid.NewV4().String(),
			Spec: &storage.NetworkPolicySpec{
				PolicyTypes: []storage.NetworkPolicyType{
					storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE,
				},
				Ingress: []*storage.NetworkPolicyIngressRule{
					{},
				},
			},
		},
		{
			Id: uuid.NewV4().String(),
			Spec: &storage.NetworkPolicySpec{
				PolicyTypes: []storage.NetworkPolicyType{
					storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE,
				},
				Egress: []*storage.NetworkPolicyEgressRule{
					{},
				},
			},
		},
	}
}

func (s *suiteImpl) policies() map[string]*storage.Policy {
	policiesMap := make(map[string]*storage.Policy)
	policies := []*storage.Policy{
		{
			Id:              uuid.NewV4().String(),
			Name:            "Sample Build time",
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
			Disabled:        false,
		},
		{
			Id:              uuid.NewV4().String(),
			Name:            "Sample Deploy time",
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
			Disabled:        false,
		},
		{
			Id:              uuid.NewV4().String(),
			Name:            "Sample Runtime time",
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
			Disabled:        false,
		},
	}

	for _, p := range policies {
		policiesMap[p.Name] = p
	}

	return policiesMap
}

func (s *suiteImpl) nodes() []*storage.Node {
	return []*storage.Node{
		{
			Id: uuid.NewV4().String(),
		},
		{
			Id: uuid.NewV4().String(),
		},
	}
}

func toMap(in []*storage.NetworkPolicy) map[string]*storage.NetworkPolicy {
	merp := make(map[string]*storage.NetworkPolicy, len(in))
	for _, np := range in {
		merp[np.GetId()] = np
	}
	return merp
}
