package datastore

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	alertMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	clusterMocks "github.com/stackrox/rox/central/cluster/store/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	nodeMocks "github.com/stackrox/rox/central/node/store/mocks"
	secretMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

const fakeClusterID = "FAKECLUSTERID"

func TestClusterDataStore(t *testing.T) {
	suite.Run(t, new(ClusterDataStoreTestSuite))
}

type ClusterDataStoreTestSuite struct {
	suite.Suite

	clusters         *clusterMocks.MockStore
	clusterDataStore DataStore

	mockCtrl *gomock.Controller
}

func (suite *ClusterDataStoreTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.clusters = clusterMocks.NewMockStore(suite.mockCtrl)
	deployments := deploymentMocks.NewMockDataStore(suite.mockCtrl)
	alerts := alertMocks.NewMockDataStore(suite.mockCtrl)
	nodes := nodeMocks.NewMockGlobalStore(suite.mockCtrl)
	secrets := secretMocks.NewMockDataStore(suite.mockCtrl)

	suite.clusterDataStore = New(suite.clusters, alerts, deployments, nodes, secrets, nil)
}

func (suite *ClusterDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

//// Test that when the cluster we try to remove does not exist, we return an error.
func (suite *ClusterDataStoreTestSuite) TestHandlesClusterDoesNotExist() {
	// Return false for the cluster not existing.
	suite.clusters.EXPECT().GetCluster(fakeClusterID).Return((*storage.Cluster)(nil), false, nil)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster(fakeClusterID)
	suite.Error(err, "expected an error since the cluster did not exist")
}

// Test that when we cannot fetch a cluster, we return the error from the DB.
func (suite *ClusterDataStoreTestSuite) TestHandlesErrorGettingCluster() {
	// Return an error trying to fetch the cluster.
	expectedErr := fmt.Errorf("issues need tissues")
	suite.clusters.EXPECT().GetCluster(fakeClusterID).Return((*storage.Cluster)(nil), true, expectedErr)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster(fakeClusterID)
	suite.Equal(expectedErr, err)
}
