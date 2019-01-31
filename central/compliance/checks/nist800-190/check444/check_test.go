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

	testPolicies := []*storage.Policy{
		{
			Id: uuid.NewV4().String(),
			LifecycleStages: []storage.LifecycleStage{
				storage.LifecycleStage_RUNTIME,
			},
		},
		{
			Id: uuid.NewV4().String(),
			LifecycleStages: []storage.LifecycleStage{
				storage.LifecycleStage_DEPLOY,
			},
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Policies().AnyTimes().Return(toMap(testPolicies))

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments)
	err = run.Run(context.Background(), domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	s.NotNil(checkResults)
	s.Len(checkResults.Evidence(), 1)
	s.Equal(framework.PassStatus, checkResults.Evidence()[0].Status)
}

func (s *suiteImpl) TestFail() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	testNodes := s.nodes()

	testPolicies := []*storage.Policy{
		{
			Id: uuid.NewV4().String(),
			LifecycleStages: []storage.LifecycleStage{
				storage.LifecycleStage_DEPLOY,
			},
		},
		{
			Id: uuid.NewV4().String(),
			LifecycleStages: []storage.LifecycleStage{
				storage.LifecycleStage_DEPLOY,
			},
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Policies().AnyTimes().Return(toMap(testPolicies))

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments)
	err = run.Run(context.Background(), domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	s.NotNil(checkResults)
	s.Len(checkResults.Evidence(), 1)
	s.Equal(framework.FailStatus, checkResults.Evidence()[0].Status)
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

func toMap(in []*storage.Policy) map[string]*storage.Policy {
	merp := make(map[string]*storage.Policy, len(in))
	for _, np := range in {
		merp[np.GetId()] = np
	}
	return merp
}
