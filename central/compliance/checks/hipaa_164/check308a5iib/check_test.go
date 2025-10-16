package check308a5iib

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

func (s *suiteImpl) TestFail() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()
	testNodes := s.nodes()
	deployment := &storage.Deployment{}
	deployment.SetId(uuid.NewV4().String())
	testDeployments := []*storage.Deployment{
		deployment,
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Policies().AnyTimes().Return(nil)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[checkID]
	s.NotNil(checkResults)

	s.NoError(checkResults.Error())
	s.Len(checkResults.Evidence(), 3)
	s.Equal(framework.FailStatus, checkResults.Evidence()[0].Status)
	s.Equal(framework.FailStatus, checkResults.Evidence()[1].Status)
	s.Equal(framework.FailStatus, checkResults.Evidence()[2].Status)
}

func (s *suiteImpl) TestPass() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	deployment := &storage.Deployment{}
	deployment.SetId(uuid.NewV4().String())
	testDeployments := []*storage.Deployment{
		deployment,
	}

	testNodes := s.nodes()
	testPolicies := s.policies()

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Policies().AnyTimes().Return(testPolicies)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[checkID]
	s.NotNil(checkResults)

	s.NoError(checkResults.Error())
	s.Len(checkResults.Evidence(), 3)
	s.Equal(framework.PassStatus, checkResults.Evidence()[0].Status)
	s.Equal(framework.PassStatus, checkResults.Evidence()[1].Status)
	s.Equal(framework.PassStatus, checkResults.Evidence()[2].Status)
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
	cluster := &storage.Cluster{}
	cluster.SetId(uuid.NewV4().String())
	return cluster
}

func (s *suiteImpl) policies() map[string]*storage.Policy {
	policiesMap := make(map[string]*storage.Policy)
	policy := &storage.Policy{}
	policy.SetId(uuid.NewV4().String())
	policy.SetName("Sample Build time")
	policy.SetLifecycleStages([]storage.LifecycleStage{storage.LifecycleStage_BUILD})
	policy.SetDisabled(false)
	policy2 := &storage.Policy{}
	policy2.SetId(uuid.NewV4().String())
	policy2.SetName("Sample Deploy time")
	policy2.SetLifecycleStages([]storage.LifecycleStage{storage.LifecycleStage_DEPLOY})
	policy2.SetDisabled(false)
	policy3 := &storage.Policy{}
	policy3.SetId(uuid.NewV4().String())
	policy3.SetName("Sample Runtime time")
	policy3.SetLifecycleStages([]storage.LifecycleStage{storage.LifecycleStage_RUNTIME})
	policy3.SetDisabled(false)
	policies := []*storage.Policy{
		policy,
		policy2,
		policy3,
	}

	for _, p := range policies {
		policiesMap[p.GetName()] = p
	}

	return policiesMap
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
