package check306e

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
	imageIntegrations := []*storage.ImageIntegration{
		{
			Id: "ii1",
			Categories: []storage.ImageIntegrationCategory{
				storage.ImageIntegrationCategory_REGISTRY,
			},
		},
	}
	images := []*storage.ListImage{
		{
			Name: "nginx",
			SetFixable: &storage.ListImage_FixableCves{
				FixableCves: 1,
			},
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().ImageIntegrations().AnyTimes().Return(imageIntegrations)
	data.EXPECT().Images().AnyTimes().Return(images)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, nil, nil, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[checkID]
	s.NotNil(checkResults)

	s.NoError(checkResults.Error())
	s.Len(checkResults.Evidence(), 2)
	s.Equal(framework.FailStatus, checkResults.Evidence()[0].Status)
	s.Equal(framework.FailStatus, checkResults.Evidence()[1].Status)
}

func (s *suiteImpl) TestPass() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()
	imageIntegrations := []*storage.ImageIntegration{
		{
			Id: "ii1",
			Categories: []storage.ImageIntegrationCategory{
				storage.ImageIntegrationCategory_REGISTRY,
			},
		},
		{
			Id: "ii2",
			Categories: []storage.ImageIntegrationCategory{
				storage.ImageIntegrationCategory_SCANNER,
			},
		},
	}
	images := []*storage.ListImage{
		{
			Name: "nginx",
			SetCves: &storage.ListImage_Cves{
				Cves: 0,
			},
			SetFixable: &storage.ListImage_FixableCves{
				FixableCves: 0,
			},
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().ImageIntegrations().AnyTimes().Return(imageIntegrations)
	data.EXPECT().Images().AnyTimes().Return(images)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, nil, nil, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[checkID]
	s.NotNil(checkResults)

	s.NoError(checkResults.Error())
	s.Len(checkResults.Evidence(), 2)
	s.Equal(framework.PassStatus, checkResults.Evidence()[0].Status)
	s.Equal(framework.PassStatus, checkResults.Evidence()[1].Status)
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
