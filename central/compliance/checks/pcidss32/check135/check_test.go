package check135

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

func (s *suiteImpl) TestFail() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()
	testDeployments := s.deployments()

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, nil, testDeployments[1:], nil, nil)
	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
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
	testDeployments := s.deployments()

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, nil, testDeployments[:1], nil, nil)
	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
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

// First deployment will pass, second deployment will fail.
func (s *suiteImpl) deployments() []*storage.Deployment {
	return []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
			Ports: []*storage.PortConfig{
				{
					Protocol: "TCP",
				},
			},
		},
		{
			Id: uuid.NewV4().String(),
			Ports: []*storage.PortConfig{
				{
					Protocol: "UDP",
				},
			},
		},
	}
}
