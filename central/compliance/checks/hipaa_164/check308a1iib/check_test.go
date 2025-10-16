package check308a1iib

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
	ii := &storage.ImageIntegration{}
	ii.SetId("ii1")
	ii.SetCategories([]storage.ImageIntegrationCategory{
		storage.ImageIntegrationCategory_REGISTRY,
	})
	imageIntegrations := []*storage.ImageIntegration{
		ii,
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().ImageIntegrations().AnyTimes().Return(imageIntegrations)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, nil, nil, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[checkID]
	s.NotNil(checkResults)

	s.NoError(checkResults.Error())
	s.Len(checkResults.Evidence(), 1)
	s.Equal(framework.FailStatus, checkResults.Evidence()[0].Status)
}

func (s *suiteImpl) TestPass() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()
	ii := &storage.ImageIntegration{}
	ii.SetId("ii1")
	ii.SetCategories([]storage.ImageIntegrationCategory{
		storage.ImageIntegrationCategory_REGISTRY,
	})
	ii2 := &storage.ImageIntegration{}
	ii2.SetId("ii2")
	ii2.SetCategories([]storage.ImageIntegrationCategory{
		storage.ImageIntegrationCategory_SCANNER,
	})
	imageIntegrations := []*storage.ImageIntegration{
		ii,
		ii2,
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().ImageIntegrations().AnyTimes().Return(imageIntegrations)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, nil, nil, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[checkID]
	s.NotNil(checkResults)

	s.NoError(checkResults.Error())
	s.Len(checkResults.Evidence(), 1)
	s.Equal(framework.PassStatus, checkResults.Evidence()[0].Status)
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
