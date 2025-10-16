package check444

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

func (s *suiteImpl) TestPass() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	testNodes := s.nodes()

	policy := &storage.Policy{}
	policy.SetId(uuid.NewV4().String())
	policy.SetLifecycleStages([]storage.LifecycleStage{
		storage.LifecycleStage_RUNTIME,
	})
	policy2 := &storage.Policy{}
	policy2.SetId(uuid.NewV4().String())
	policy2.SetLifecycleStages([]storage.LifecycleStage{
		storage.LifecycleStage_DEPLOY,
	})
	testPolicies := []*storage.Policy{
		policy,
		policy2,
	}

	testDeployments := []*storage.Deployment{
		storage.Deployment_builder{
			Id:   uuid.NewV4().String(),
			Name: "Foo",
			Containers: []*storage.Container{
				storage.Container_builder{
					Name: "container-foo",
					SecurityContext: storage.SecurityContext_builder{
						ReadOnlyRootFilesystem: true,
					}.Build(),
				}.Build(),
			},
		}.Build(),
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Policies().AnyTimes().Return(toMap(testPolicies))
	data.EXPECT().Deployments().AnyTimes().Return(toMapDeployments(testDeployments))

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	s.NotNil(checkResults)
	s.Len(checkResults.Evidence(), 1)
	s.Equal(framework.PassStatus, checkResults.Evidence()[0].Status)

	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		s.NoError(deploymentResults.Error())
		if s.Len(deploymentResults.Evidence(), 1) {
			s.Equal(framework.PassStatus, deploymentResults.Evidence()[0].Status)
		}
	}
}

func (s *suiteImpl) TestFail() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	testNodes := s.nodes()

	policy := &storage.Policy{}
	policy.SetId(uuid.NewV4().String())
	policy.SetLifecycleStages([]storage.LifecycleStage{
		storage.LifecycleStage_DEPLOY,
	})
	policy2 := &storage.Policy{}
	policy2.SetId(uuid.NewV4().String())
	policy2.SetLifecycleStages([]storage.LifecycleStage{
		storage.LifecycleStage_DEPLOY,
	})
	testPolicies := []*storage.Policy{
		policy,
		policy2,
	}

	testDeployments := []*storage.Deployment{
		storage.Deployment_builder{
			Id:   uuid.NewV4().String(),
			Name: "Foo",
			Containers: []*storage.Container{
				storage.Container_builder{
					Name: "container-foo",
					SecurityContext: storage.SecurityContext_builder{
						ReadOnlyRootFilesystem: false,
					}.Build(),
				}.Build(),
			},
		}.Build(),
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Policies().AnyTimes().Return(toMap(testPolicies))
	data.EXPECT().Deployments().AnyTimes().Return(toMapDeployments(testDeployments))

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	s.NotNil(checkResults)
	s.Len(checkResults.Evidence(), 1)
	s.Equal(framework.FailStatus, checkResults.Evidence()[0].Status)

	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		s.NoError(deploymentResults.Error())
		if s.Len(deploymentResults.Evidence(), 1) {
			s.Equal(framework.FailStatus, deploymentResults.Evidence()[0].Status)
		}
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

func toMap(in []*storage.Policy) map[string]*storage.Policy {
	merp := make(map[string]*storage.Policy, len(in))
	for _, np := range in {
		merp[np.GetId()] = np
	}
	return merp
}

func toMapDeployments(in []*storage.Deployment) map[string]*storage.Deployment {
	merp := make(map[string]*storage.Deployment, len(in))
	for _, np := range in {
		merp[np.GetId()] = np
	}
	return merp
}
