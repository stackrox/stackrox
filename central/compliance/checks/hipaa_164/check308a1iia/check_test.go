package check308a1iia

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
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
	listImage := &storage.ListImage{}
	listImage.SetName("nginx")
	listImage.ClearSetCves()
	listImage.SetFixableCves(1)
	images := []*storage.ListImage{
		listImage,
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(s.cluster())
	data.EXPECT().ImageIntegrations().AnyTimes().Return(imageIntegrations)
	data.EXPECT().Images().AnyTimes().Return(images)
	data.EXPECT().HasProcessIndicators().AnyTimes().Return(false)
	data.EXPECT().NetworkFlowsWithDeploymentDst().AnyTimes().Return(nil)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, nil, nil, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[checkID]
	s.NotNil(checkResults)

	s.NoError(checkResults.Error())
	s.Len(checkResults.Evidence(), 3)
	s.Equal(framework.FailStatus, checkResults.Evidence()[0].Status)
	s.Equal(framework.FailStatus, checkResults.Evidence()[1].Status)
	s.Equal(framework.FailStatus, checkResults.Evidence()[1].Status)
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
	listImage := &storage.ListImage{}
	listImage.SetName("nginx")
	listImage.Set_Cves(0)
	listImage.SetFixableCves(0)
	images := []*storage.ListImage{
		listImage,
	}
	nf := &storage.NetworkFlow{}
	nf.SetLastSeenTimestamp(protocompat.TimestampNow())
	flows := []*storage.NetworkFlow{
		nf,
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(s.cluster())
	data.EXPECT().ImageIntegrations().AnyTimes().Return(imageIntegrations)
	data.EXPECT().Images().AnyTimes().Return(images)
	data.EXPECT().HasProcessIndicators().AnyTimes().Return(true)
	data.EXPECT().NetworkFlowsWithDeploymentDst().AnyTimes().Return(flows)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, nil, nil, nil)
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
	cluster.SetCollectionMethod(storage.CollectionMethod_EBPF)
	return cluster
}
