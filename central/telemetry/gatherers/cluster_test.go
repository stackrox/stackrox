package gatherers

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	nodeMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	mockClusters = []*storage.Cluster{
		{
			Id:             "abc",
			Name:           "123",
			MainImage:      "Joseph Rules",
			CollectorImage: "098",
			Status: &storage.ClusterStatus{
				SensorVersion: "555",
				ProviderMetadata: &storage.ProviderMetadata{
					Provider: &storage.ProviderMetadata_Google{
						Google: &storage.GoogleProviderMetadata{},
					},
				},
				OrchestratorMetadata: &storage.OrchestratorMetadata{
					Version: "333",
				},
			},
			HealthStatus: &storage.ClusterHealthStatus{
				LastContact: &types.Timestamp{Seconds: 300},
			},
		},
	}
)

func TestClusterGatherer(t *testing.T) {
	suite.Run(t, new(clusterGathererTestSuite))
}

type clusterGathererTestSuite struct {
	suite.Suite

	gatherer                *ClusterGatherer
	mockClusterDatastore    *clusterMocks.MockDataStore
	mockNodeDatastore       *nodeMocks.MockDataStore
	mockNamespaceDatastore  *namespaceMocks.MockDataStore
	mockConnectionManager   *connectionMocks.MockManager
	mockDeploymentDatastore *deploymentMocks.MockDataStore
	mockCtrl                *gomock.Controller
}

func (s *clusterGathererTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockClusterDatastore = clusterMocks.NewMockDataStore(s.mockCtrl)
	s.mockNodeDatastore = nodeMocks.NewMockDataStore(s.mockCtrl)
	s.mockNamespaceDatastore = namespaceMocks.NewMockDataStore(s.mockCtrl)
	s.mockConnectionManager = connectionMocks.NewMockManager(s.mockCtrl)
	s.mockDeploymentDatastore = deploymentMocks.NewMockDataStore(s.mockCtrl)
	s.gatherer = newClusterGatherer(s.mockClusterDatastore, s.mockNodeDatastore, s.mockNamespaceDatastore, s.mockConnectionManager, s.mockDeploymentDatastore)
}

func (s *clusterGathererTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

// Test just ensures that gathering doesn't panic.  I didn't want to test more specific data because these gatherers
// don't have much business logic.  Testing that each field was set is essentially a change detector test.
func (s *clusterGathererTestSuite) TestGather() {
	s.mockClusterDatastore.EXPECT().GetClusters(gomock.Any()).Return(mockClusters, nil)
	s.mockNamespaceDatastore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return(nil, nil)
	s.mockNodeDatastore.EXPECT().SearchRawNodes(gomock.Any(), gomock.Any()).Return(nil, nil)
	s.mockConnectionManager.EXPECT().GetActiveConnections().Return(nil)
	clusters := s.gatherer.Gather(context.Background(), true)
	s.Len(clusters, 1)
	cluster := clusters[0]
	mockCluster := mockClusters[0]
	s.Equal(mockCluster.GetId(), cluster.ID)
	s.Equal("Google", cluster.CloudProvider)
}
