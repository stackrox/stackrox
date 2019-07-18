package manager_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	clusterDatastoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/compliance"
	complianceDataMocks "github.com/stackrox/rox/central/compliance/data/mocks"
	complianceDSMocks "github.com/stackrox/rox/central/compliance/datastore/mocks"
	"github.com/stackrox/rox/central/compliance/manager"
	complianceMgrMocks "github.com/stackrox/rox/central/compliance/manager/mocks"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	deploymentDatastoreMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	nodeDatastoreMocks "github.com/stackrox/rox/central/node/globaldatastore/mocks"
	scrapeMocks "github.com/stackrox/rox/central/scrape/factory/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

type managerTestSuite struct {
	suite.Suite

	manager manager.ComplianceManager
	testCtx context.Context

	mockCtrl            *gomock.Controller
	standardRegistry    *standards.Registry
	mockScheduleStore   *complianceMgrMocks.MockScheduleStore
	mockClusterStore    *clusterDatastoreMocks.MockDataStore
	mockNodeStore       *nodeDatastoreMocks.MockGlobalDataStore
	mockDeploymentStore *deploymentDatastoreMocks.MockDataStore
	mockDataRepoFactory *complianceDataMocks.MockRepositoryFactory
	mockScrapeFactory   *scrapeMocks.MockScrapeFactory
	mockResultsStore    *complianceDSMocks.MockDataStore
}

func TestManager(t *testing.T) {
	suite.Run(t, new(managerTestSuite))
}

func (s *managerTestSuite) TestExpandSelection_OneOne() {
	pairs, err := s.manager.ExpandSelection(s.testCtx, "cluster1", "standard1")
	s.NoError(err)
	s.ElementsMatch(pairs, []compliance.ClusterStandardPair{
		{ClusterID: "cluster1", StandardID: "standard1"},
	})
}

func (s *managerTestSuite) TestExpandSelection_AllOne_OK() {
	s.mockClusterStore.EXPECT().GetClusters(s.testCtx).Return([]*storage.Cluster{
		{
			Id: "cluster1",
		},
		{
			Id: "cluster2",
		},
	}, nil)
	pairs, err := s.manager.ExpandSelection(s.testCtx, manager.Wildcard, "standard1")
	s.NoError(err)
	s.ElementsMatch(pairs, []compliance.ClusterStandardPair{
		{ClusterID: "cluster1", StandardID: "standard1"},
		{ClusterID: "cluster2", StandardID: "standard1"},
	})
}

func (s *managerTestSuite) TestExpandSelection_AllOne_GetClustersError() {
	s.mockClusterStore.EXPECT().GetClusters(s.testCtx).Return(nil, errors.New("some error"))
	_, err := s.manager.ExpandSelection(s.testCtx, manager.Wildcard, "standard1")
	s.Error(err)
}

func (s *managerTestSuite) TestExpandSelection_OneAll_OK() {
	var err error
	s.standardRegistry, err = standards.NewRegistry(nil, nil,
		metadata.Standard{ID: "standard1"},
		metadata.Standard{ID: "standard2"},
	)
	s.Require().NoError(err)
	s.manager, err = manager.NewManager(s.standardRegistry, s.mockScheduleStore, s.mockClusterStore, s.mockNodeStore, s.mockDeploymentStore, s.mockDataRepoFactory, s.mockScrapeFactory, s.mockResultsStore)
	s.Require().NoError(err)
	pairs, err := s.manager.ExpandSelection(s.testCtx, "cluster1", manager.Wildcard)
	s.NoError(err)
	s.ElementsMatch(pairs, []compliance.ClusterStandardPair{
		{ClusterID: "cluster1", StandardID: "standard1"},
		{ClusterID: "cluster1", StandardID: "standard2"},
	})
}

func (s *managerTestSuite) TestExpandSelection_AllAll_OK() {
	s.mockClusterStore.EXPECT().GetClusters(s.testCtx).Return([]*storage.Cluster{
		{Id: "cluster1"},
		{Id: "cluster2"},
	}, nil)
	var err error
	s.standardRegistry, err = standards.NewRegistry(nil, nil,
		metadata.Standard{ID: "standard1"},
		metadata.Standard{ID: "standard2"},
	)
	s.Require().NoError(err)
	s.manager, err = manager.NewManager(s.standardRegistry, s.mockScheduleStore, s.mockClusterStore, s.mockNodeStore, s.mockDeploymentStore, s.mockDataRepoFactory, s.mockScrapeFactory, s.mockResultsStore)
	s.Require().NoError(err)
	pairs, err := s.manager.ExpandSelection(s.testCtx, manager.Wildcard, manager.Wildcard)
	s.NoError(err)
	s.ElementsMatch(pairs, []compliance.ClusterStandardPair{
		{ClusterID: "cluster1", StandardID: "standard1"},
		{ClusterID: "cluster1", StandardID: "standard2"},
		{ClusterID: "cluster2", StandardID: "standard1"},
		{ClusterID: "cluster2", StandardID: "standard2"},
	})
}

func (s *managerTestSuite) SetupTest() {
	s.testCtx = context.Background()
	s.mockCtrl = gomock.NewController(s.T())
	var err error
	s.standardRegistry, err = standards.NewRegistry(nil, nil)
	s.Require().NoError(err)
	s.mockScheduleStore = complianceMgrMocks.NewMockScheduleStore(s.mockCtrl)
	s.mockClusterStore = clusterDatastoreMocks.NewMockDataStore(s.mockCtrl)
	s.mockNodeStore = nodeDatastoreMocks.NewMockGlobalDataStore(s.mockCtrl)
	s.mockDeploymentStore = deploymentDatastoreMocks.NewMockDataStore(s.mockCtrl)
	s.mockScrapeFactory = scrapeMocks.NewMockScrapeFactory(s.mockCtrl)
	s.mockResultsStore = complianceDSMocks.NewMockDataStore(s.mockCtrl)

	s.mockScheduleStore.EXPECT().ListSchedules().AnyTimes().Return(nil, nil)
	s.manager, err = manager.NewManager(s.standardRegistry, s.mockScheduleStore, s.mockClusterStore, s.mockNodeStore, s.mockDeploymentStore, s.mockDataRepoFactory, s.mockScrapeFactory, s.mockResultsStore)
	s.Require().NoError(err)
}

func (s *managerTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}
