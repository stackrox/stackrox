package check221

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/storage"
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

	deployments := []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
			Containers: []*storage.Container{
				{
					Name: "container1",
				},
			},
		},
	}

	indicators := []*storage.ProcessIndicator{
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  deployments[0].GetId(),
			ContainerName: deployments[0].GetContainers()[0].GetName(),
			Signal: &storage.ProcessSignal{
				Pid:  15,
				Name: "nginx",
			},
		},
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  deployments[0].GetId(),
			ContainerName: deployments[0].GetContainers()[0].GetName(),
			Signal: &storage.ProcessSignal{
				Pid:  32,
				Name: "zsh",
			},
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().ProcessIndicators().AnyTimes().Return(indicators)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, nil, deployments)
	err = run.Run(context.Background(), domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[checkID]
	s.NotNil(checkResults)

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

	deployments := []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
			Containers: []*storage.Container{
				{
					Name: "container1",
				},
			},
		},
	}

	indicators := []*storage.ProcessIndicator{
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  deployments[0].GetId(),
			ContainerName: deployments[0].GetContainers()[0].GetName(),
			Signal: &storage.ProcessSignal{
				Pid:  15,
				Name: "nginx",
			},
		},
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  deployments[0].GetId(),
			ContainerName: deployments[0].GetContainers()[0].GetName(),
			Signal: &storage.ProcessSignal{
				Pid:  32,
				Name: "nginx",
			},
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().ProcessIndicators().AnyTimes().Return(indicators)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, nil, deployments)
	err = run.Run(context.Background(), domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[checkID]
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
	check := registry.Lookup(checkID)
	s.NotNil(check)
	return check
}

func (s *suiteImpl) cluster() *storage.Cluster {
	return &storage.Cluster{
		Id: uuid.NewV4().String(),
	}
}
