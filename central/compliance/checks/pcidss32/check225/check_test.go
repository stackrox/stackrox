package check225

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

func (s *suiteImpl) TestUnusedPorts() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	// Both deployments have port 3 exposed.
	deployments := []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
			Ports: []*storage.PortConfig{
				{
					ContainerPort: 3,
				},
			},
		},
		{
			Id: uuid.NewV4().String(),
			Ports: []*storage.PortConfig{
				{
					ContainerPort: 3,
				},
			},
		},
	}

	// No network flows occuring on port 3.
	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 2,
				DstEntity: &storage.NetworkEntityInfo{
					Id: deployments[0].GetId(),
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Id: deployments[1].GetId(),
				},
			},
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().NetworkFlowsWithDeploymentDst().AnyTimes().Return(flows)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, nil, deployments, nil, nil)
	err = run.Run(context.Background(), "standard", domain, data)
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

	// Both deployments have port 3 exposed.
	deployments := []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
			Ports: []*storage.PortConfig{
				{
					ContainerPort: 3,
				},
			},
		},
		{
			Id: uuid.NewV4().String(),
			Ports: []*storage.PortConfig{
				{
					ContainerPort: 3,
				},
			},
		},
	}

	// Both deployments talk to each other on port 3.
	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 3,
				DstEntity: &storage.NetworkEntityInfo{
					Id: deployments[0].GetId(),
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Id: deployments[1].GetId(),
				},
			},
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 3,
				DstEntity: &storage.NetworkEntityInfo{
					Id: deployments[1].GetId(),
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Id: deployments[0].GetId(),
				},
			},
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().NetworkFlowsWithDeploymentDst().AnyTimes().Return(flows)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, nil, deployments, nil, nil)
	err = run.Run(context.Background(), "standard", domain, data)
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
