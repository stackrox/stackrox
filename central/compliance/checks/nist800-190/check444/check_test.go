package check444

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

var (
	testDeployments = []*storage.Deployment{
		{
			Id:   uuid.NewV4().String(),
			Name: "foo",
		},
		{
			Id:   uuid.NewV4().String(),
			Name: "boo",
		},
	}
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

func (s *suiteImpl) TestPass() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	testNodes := s.nodes()

	testAlerts := []*storage.ListAlert{
		{
			Deployment: &storage.ListAlertDeployment{
				Id:   testDeployments[0].GetId(),
				Name: "foo",
			},
			LifecycleStage:   storage.LifecycleStage_RUNTIME,
			EnforcementCount: 1,
		},
		{
			Deployment: &storage.ListAlertDeployment{
				Id:   testDeployments[1].GetId(),
				Name: "boo",
			},
			LifecycleStage:   storage.LifecycleStage_RUNTIME,
			EnforcementCount: 1,
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Alerts().AnyTimes().Return(testAlerts)
	data.EXPECT().Deployments().AnyTimes().Return(toMap(testDeployments))

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments)
	err = run.Run(context.Background(), domain, data)
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

func (s *suiteImpl) TestFail() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	testNodes := s.nodes()

	testAlerts := []*storage.ListAlert{
		{
			Deployment: &storage.ListAlertDeployment{
				Id:   testDeployments[0].GetId(),
				Name: "foo",
			},
			LifecycleStage:   storage.LifecycleStage_RUNTIME,
			EnforcementCount: 0,
		},
		{
			Deployment: &storage.ListAlertDeployment{
				Id:   testDeployments[1].GetId(),
				Name: "boo",
			},
			LifecycleStage:   storage.LifecycleStage_RUNTIME,
			EnforcementCount: 0,
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Alerts().AnyTimes().Return(testAlerts)
	data.EXPECT().Deployments().AnyTimes().Return(toMap(testDeployments))

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments)
	err = run.Run(context.Background(), domain, data)
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

// Helper functions for test data.
//////////////////////////////////

func (s *suiteImpl) verifyCheckRegistered() framework.Check {
	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	s.NotNil(check)
	return check
}

func (s *suiteImpl) cluster() *storage.Cluster {
	return &storage.Cluster{
		Id: uuid.NewV4().String(),
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

func toMap(in []*storage.Deployment) map[string]*storage.Deployment {
	merp := make(map[string]*storage.Deployment, len(in))
	for _, np := range in {
		merp[np.GetId()] = np
	}
	return merp
}
