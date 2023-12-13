package check134

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

type testCase struct {
	cluster                      *storage.Cluster
	nodes                        []*storage.Node
	deployments                  []*storage.Deployment
	networkPolicies              []*storage.NetworkPolicy
	deploymentsToNetworkPolicies map[string][]*storage.NetworkPolicy
	expectedStatus               framework.Status
}

func (s *suiteImpl) TestHostNetwork() {
	tc := testCase{}

	tc.cluster = s.cluster()
	tc.nodes = s.nodes()
	tc.networkPolicies = s.networkPolicies()

	// host network enabled (should fail)
	tc.deployments = []*storage.Deployment{
		{
			Id:          uuid.NewV4().String(),
			HostNetwork: true,
		},
	}

	// ingress and egress networkpolicies enabled
	tc.deploymentsToNetworkPolicies = map[string][]*storage.NetworkPolicy{
		tc.deployments[0].GetId(): {tc.networkPolicies[0], tc.networkPolicies[1]},
	}

	tc.expectedStatus = framework.FailStatus
	s.checkTestCase(&tc)
}

func (s *suiteImpl) TestEgress() {
	tc := testCase{}

	tc.cluster = s.cluster()
	tc.nodes = s.nodes()
	tc.networkPolicies = s.networkPolicies()

	tc.deployments = []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
		},
	}

	// only egress networkpolicies enabled
	tc.deploymentsToNetworkPolicies = map[string][]*storage.NetworkPolicy{
		tc.deployments[0].GetId(): {tc.networkPolicies[1]},
	}

	tc.expectedStatus = framework.PassStatus
	s.checkTestCase(&tc)
}

func (s *suiteImpl) TestIngress() {
	tc := testCase{}

	tc.cluster = s.cluster()
	tc.nodes = s.nodes()
	tc.networkPolicies = s.networkPolicies()

	tc.deployments = []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
		},
	}

	// only ingress networkpolicies enabled
	tc.deploymentsToNetworkPolicies = map[string][]*storage.NetworkPolicy{
		tc.deployments[0].GetId(): {tc.networkPolicies[0]},
	}

	tc.expectedStatus = framework.FailStatus
	s.checkTestCase(&tc)
}

func (s *suiteImpl) TestKubeSystem() {
	tc := testCase{}

	tc.cluster = s.cluster()
	tc.nodes = s.nodes()
	tc.networkPolicies = s.networkPolicies()

	tc.deployments = []*storage.Deployment{
		{
			Id:          uuid.NewV4().String(),
			HostNetwork: true,
			Namespace:   "kube-system",
		},
	}

	tc.deploymentsToNetworkPolicies = map[string][]*storage.NetworkPolicy{
		tc.deployments[0].GetId(): {},
	}

	tc.expectedStatus = framework.SkipStatus
	s.checkTestCase(&tc)
}

func (s *suiteImpl) TestPass() {
	tc := testCase{}

	tc.cluster = s.cluster()
	tc.nodes = s.nodes()
	tc.networkPolicies = s.networkPolicies()

	tc.deployments = []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
		},
		{
			Id: uuid.NewV4().String(),
		},
	}

	tc.deploymentsToNetworkPolicies = map[string][]*storage.NetworkPolicy{
		tc.deployments[0].GetId(): {tc.networkPolicies[0], tc.networkPolicies[1]},
		tc.deployments[1].GetId(): {tc.networkPolicies[0], tc.networkPolicies[1]},
	}

	tc.expectedStatus = framework.PassStatus
	s.checkTestCase(&tc)
}

// Helper functions for test data.
//////////////////////////////////

func (s *suiteImpl) verifyCheckRegistered() framework.Check {
	registry := framework.RegistrySingleton()
	check := registry.Lookup(checkID)
	s.NotNil(check)
	return check
}

func (s *suiteImpl) checkTestCase(tc *testCase) {

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().DeploymentsToNetworkPolicies().AnyTimes().Return(tc.deploymentsToNetworkPolicies)

	check := s.verifyCheckRegistered()
	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(tc.cluster, tc.nodes, tc.deployments, nil, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[checkID]
	s.NotNil(checkResults)

	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		s.NoError(deploymentResults.Error())
		s.Len(deploymentResults.Evidence(), 1)
		s.Equal(tc.expectedStatus, deploymentResults.Evidence()[0].Status)
	}

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
