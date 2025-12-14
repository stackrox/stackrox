package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/deployment/cache"
	graphConfigMocks "github.com/stackrox/rox/central/networkgraph/config/datastore/mocks"
	networkTreeMocks "github.com/stackrox/rox/central/networkgraph/entity/networktree/mocks"
	storeMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestNetworkFlowClusterDataStore(t *testing.T) {
	suite.Run(t, new(networkFlowClusterDataStoreTestSuite))
}

type networkFlowClusterDataStoreTestSuite struct {
	suite.Suite

	mockCtrl                *gomock.Controller
	mockStorage             *storeMocks.MockClusterStore
	mockGraphConfig         *graphConfigMocks.MockDataStore
	mockNetworkTreeMgr      *networkTreeMocks.MockManager
	deletedDeploymentsCache cache.DeletedDeployments

	dataStore ClusterDataStore

	noAccessCtx  context.Context
	allAccessCtx context.Context
}

func (s *networkFlowClusterDataStoreTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockStorage = storeMocks.NewMockClusterStore(s.mockCtrl)
	s.mockGraphConfig = graphConfigMocks.NewMockDataStore(s.mockCtrl)
	s.mockNetworkTreeMgr = networkTreeMocks.NewMockManager(s.mockCtrl)
	s.deletedDeploymentsCache = cache.DeletedDeploymentsSingleton()

	s.dataStore = NewClusterDataStore(
		s.mockStorage,
		s.mockGraphConfig,
		s.mockNetworkTreeMgr,
		s.deletedDeploymentsCache,
	)

	s.noAccessCtx = sac.WithNoAccess(context.Background())
	s.allAccessCtx = sac.WithAllAccess(context.Background())
}

func (s *networkFlowClusterDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *networkFlowClusterDataStoreTestSuite) TestRemoveFlowStore() {
	clusterID := fixtureconsts.Cluster1
	s.Run("NoAccess", func() {
		// When context has no access, the function should return ErrResourceAccessDenied
		// and should not call the storage's RemoveFlowStore method
		err := s.dataStore.RemoveFlowStore(s.noAccessCtx, clusterID)

		s.Error(err)
		s.Equal(sac.ErrResourceAccessDenied, err)
	})
	s.Run("AllAccess", func() {
		s.Run("StoreSuccess", func() {
			// When context has all access, the function should call the storage's RemoveFlowStore method
			s.mockStorage.EXPECT().
				RemoveFlowStore(gomock.Any(), clusterID).
				Return(nil).
				Times(1)

			err := s.dataStore.RemoveFlowStore(s.allAccessCtx, clusterID)

			s.NoError(err)
		})
		s.Run("StoreError", func() {
			expectedError := errox.InvalidArgs.New("storage error")

			// When context has all access but storage returns an error, the function should propagate it
			s.mockStorage.EXPECT().
				RemoveFlowStore(gomock.Any(), clusterID).
				Return(expectedError).
				Times(1)

			err := s.dataStore.RemoveFlowStore(s.allAccessCtx, clusterID)

			s.Error(err)
			s.Equal(expectedError, err)
		})
	})
}
