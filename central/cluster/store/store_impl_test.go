package store

import (
	"testing"
	"time"

	bolt "github.com/etcd-io/bbolt"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestClusterStore(t *testing.T) {
	suite.Run(t, new(ClusterStoreTestSuite))
}

type ClusterStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *ClusterStoreTestSuite) SetupTest() {
	suite.db = testutils.DBForSuite(suite)
	suite.store = New(suite.db)
}

func (suite *ClusterStoreTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *ClusterStoreTestSuite) hydratedCluster(cluster *storage.Cluster, status *storage.ClusterStatus, upgradeStatus *storage.ClusterUpgradeStatus) *storage.Cluster {
	clonedCluster := cluster.Clone()
	suite.Nil(status.GetUpgradeStatus())
	clonedStatus := status.Clone()
	clonedStatus.UpgradeStatus = upgradeStatus
	clonedCluster.Status = clonedStatus
	return clonedCluster
}

func (suite *ClusterStoreTestSuite) TestClusters() {
	checkin1 := time.Now()
	checkin2 := time.Now().Add(-1 * time.Hour)
	ts1, err := ptypes.TimestampProto(checkin1)
	suite.NoError(err)
	ts2, err := ptypes.TimestampProto(checkin2)
	suite.NoError(err)

	clusters := []*storage.Cluster{
		{
			Name:      "cluster1",
			MainImage: "test-dtr.example.com/main",
		},
		{
			Name:      "cluster2",
			MainImage: "docker.io/stackrox/main",
		},
	}
	statuses := []*storage.ClusterStatus{
		{
			LastContact: ts1,
			ProviderMetadata: &storage.ProviderMetadata{
				Region: "BLAH",
			},
		},
		{
			LastContact: ts2,
		},
	}

	upgradeStatuses := []*storage.ClusterUpgradeStatus{
		{
			Upgradability: storage.ClusterUpgradeStatus_UP_TO_DATE,
		},
		{
			Upgradability: storage.ClusterUpgradeStatus_AUTO_UPGRADE_POSSIBLE,
		},
	}

	// Test Add
	for _, b := range clusters {
		id, err := suite.store.AddCluster(b)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, b := range clusters {
		got, exists, err := suite.store.GetCluster(b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	for i, b := range clusters {
		suite.NoError(suite.store.UpdateClusterStatus(b.GetId(), statuses[i]))
		t, err := ptypes.TimestampFromProto(statuses[i].GetLastContact())
		suite.NoError(err)
		err = suite.store.UpdateClusterContactTimes(t, b.GetId())
		suite.NoError(err)
		suite.NoError(suite.store.UpdateClusterUpgradeStatus(b.GetId(), upgradeStatuses[i]))
	}

	for i, b := range clusters {
		got, exists, err := suite.store.GetCluster(b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, suite.hydratedCluster(b, statuses[i], upgradeStatuses[i]))
	}

	gotClusters, err := suite.store.GetClusters()
	suite.NoError(err)
	for _, gotCluster := range gotClusters {
		found := false
		for i, actualCluster := range clusters {
			if actualCluster.GetId() != gotCluster.GetId() {
				continue
			}
			found = true
			suite.Equal(gotCluster, suite.hydratedCluster(actualCluster, statuses[i], upgradeStatuses[i]))
		}
		suite.True(found)
	}

	// Test Update
	for _, b := range clusters {
		b.MainImage = b.MainImage + "/main"
	}

	for _, b := range clusters {
		suite.NoError(suite.store.UpdateCluster(b))
	}

	for i, b := range clusters {
		t, err := ptypes.TimestampFromProto(statuses[i].GetLastContact())
		suite.NoError(err)
		err = suite.store.UpdateClusterContactTimes(t, b.GetId())
		suite.NoError(err)
		suite.NoError(suite.store.UpdateClusterUpgradeStatus(b.GetId(), upgradeStatuses[i]))
		suite.NoError(suite.store.UpdateClusterStatus(b.GetId(), statuses[i]))
	}

	for i, b := range clusters {
		got, exists, err := suite.store.GetCluster(b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, suite.hydratedCluster(b, statuses[i], upgradeStatuses[i]))
	}

	// Test Count
	count, err := suite.store.CountClusters()
	suite.NoError(err)
	suite.Equal(len(clusters), count)

	// Test Remove
	for _, b := range clusters {
		suite.NoError(suite.store.RemoveCluster(b.GetId()))
	}

	for _, b := range clusters {
		_, exists, err := suite.store.GetCluster(b.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}

func (suite *ClusterStoreTestSuite) TestClusterStatusUpdatesNoRace() {
	id, err := suite.store.AddCluster(&storage.Cluster{Name: "blah"})
	suite.NoError(err)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		suite.NoError(suite.store.UpdateClusterStatus(id, &storage.ClusterStatus{SensorVersion: "BLAH"}))
	}()
	go func() {
		defer wg.Done()
		suite.NoError(suite.store.UpdateClusterUpgradeStatus(id, &storage.ClusterUpgradeStatus{Upgradability: storage.ClusterUpgradeStatus_UP_TO_DATE}))
	}()
	wg.Wait()

	got, exists, err := suite.store.GetCluster(id)
	suite.NoError(err)
	suite.True(exists)
	suite.Equal(&storage.Cluster{
		Id:   id,
		Name: "blah",
		Status: &storage.ClusterStatus{
			SensorVersion: "BLAH",
			UpgradeStatus: &storage.ClusterUpgradeStatus{
				Upgradability: storage.ClusterUpgradeStatus_UP_TO_DATE,
			},
		},
	}, got)
}
