package manager_test

import (
	"context"
	"errors"
	"testing"

	clusterDatastoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/compliance"
	complianceDataMocks "github.com/stackrox/rox/central/compliance/data/mocks"
	complianceDSMocks "github.com/stackrox/rox/central/compliance/datastore/mocks"
	"github.com/stackrox/rox/central/compliance/manager"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	deploymentDatastoreMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	nodeDatastoreMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	podDatastoreMocks "github.com/stackrox/rox/central/pod/datastore/mocks"
	scrapeMocks "github.com/stackrox/rox/central/scrape/factory/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type managerTestSuite struct {
	suite.Suite

	manager     manager.ComplianceManager
	testCtx     context.Context
	readOnlyCtx context.Context

	mockCtrl            *gomock.Controller
	standardRegistry    *standards.Registry
	mockClusterStore    *clusterDatastoreMocks.MockDataStore
	mockNodeStore       *nodeDatastoreMocks.MockDataStore
	mockDeploymentStore *deploymentDatastoreMocks.MockDataStore
	mockPodStore        *podDatastoreMocks.MockDataStore
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
	s.standardRegistry, err = standards.NewRegistry(nil,
		metadata.Standard{ID: "standard1"},
		metadata.Standard{ID: "standard2"},
	)
	s.Require().NoError(err)
	s.manager = manager.NewManager(s.standardRegistry, nil, nil, s.mockClusterStore, s.mockNodeStore, s.mockDeploymentStore, s.mockPodStore, s.mockDataRepoFactory, s.mockScrapeFactory, s.mockResultsStore)
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
	s.standardRegistry, err = standards.NewRegistry(nil,
		metadata.Standard{ID: "standard1"},
		metadata.Standard{ID: "standard2"},
	)
	s.Require().NoError(err)
	s.manager = manager.NewManager(s.standardRegistry, nil, nil, s.mockClusterStore, s.mockNodeStore, s.mockDeploymentStore, s.mockPodStore, s.mockDataRepoFactory, s.mockScrapeFactory, s.mockResultsStore)
	pairs, err := s.manager.ExpandSelection(s.testCtx, manager.Wildcard, manager.Wildcard)
	s.NoError(err)
	s.ElementsMatch(pairs, []compliance.ClusterStandardPair{
		{ClusterID: "cluster1", StandardID: "standard1"},
		{ClusterID: "cluster1", StandardID: "standard2"},
		{ClusterID: "cluster2", StandardID: "standard1"},
		{ClusterID: "cluster2", StandardID: "standard2"},
	})
	// Test with readOnly ctx
	s.mockClusterStore.EXPECT().GetClusters(s.readOnlyCtx).Return([]*storage.Cluster{
		{Id: "cluster1"},
		{Id: "cluster2"},
	}, nil)
	pairs, err = s.manager.ExpandSelection(s.readOnlyCtx, manager.Wildcard, manager.Wildcard)
	s.Equal(pairs, []compliance.ClusterStandardPair{})
	s.NoError(err)
}

func (s *managerTestSuite) SetupTest() {
	s.testCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.readOnlyCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.mockCtrl = gomock.NewController(s.T())
	var err error
	s.standardRegistry, err = standards.NewRegistry(nil)
	s.Require().NoError(err)
	s.mockClusterStore = clusterDatastoreMocks.NewMockDataStore(s.mockCtrl)
	s.mockNodeStore = nodeDatastoreMocks.NewMockDataStore(s.mockCtrl)
	s.mockDeploymentStore = deploymentDatastoreMocks.NewMockDataStore(s.mockCtrl)
	s.mockPodStore = podDatastoreMocks.NewMockDataStore(s.mockCtrl)
	s.mockScrapeFactory = scrapeMocks.NewMockScrapeFactory(s.mockCtrl)
	s.mockResultsStore = complianceDSMocks.NewMockDataStore(s.mockCtrl)

	s.manager = manager.NewManager(s.standardRegistry, nil, nil, s.mockClusterStore, s.mockNodeStore, s.mockDeploymentStore, s.mockPodStore, s.mockDataRepoFactory, s.mockScrapeFactory, s.mockResultsStore)
}

func (s *managerTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}
