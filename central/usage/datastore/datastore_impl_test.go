package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

func TestUsageDataStore(t *testing.T) {
	suite.Run(t, new(UsageDataStoreTestSuite))
}

type UsageDataStoreTestSuite struct {
	suite.Suite

	datastore DataStore
}

type testCluStore struct {
	clusters []*storage.Cluster
}

func (tcs *testCluStore) GetClusters(ctx context.Context) ([]*storage.Cluster, error) {
	if ok, err := sac.ForResource(resources.Cluster).ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}
	return tcs.clusters, nil
}

func (suite *UsageDataStoreTestSuite) SetupTest() {
	suite.datastore = New(nil, &testCluStore{
		clusters: []*storage.Cluster{{
			Id: "existingCluster1",
		}, {
			Id: "existingCluster2",
		}},
	})
}

func (suite *UsageDataStoreTestSuite) TearDownSuite() {
}

func (suite *UsageDataStoreTestSuite) TestUpdateMessage() {
	err := suite.datastore.UpdateUsage("existingCluster1", &central.ClusterMetrics{
		NodeCount:   1,
		CpuCapacity: 8,
	})
	suite.NoError(err)
}

func (suite *UsageDataStoreTestSuite) TestUpdateGetCurrent() {
	u, err := suite.datastore.GetCurrent(context.Background())
	suite.NoError(err)
	suite.Equal(int32(0), u.NumNodes)
	suite.Equal(int32(0), u.NumCores)
	err = suite.datastore.UpdateUsage("existingCluster1", &central.ClusterMetrics{
		NodeCount:   1,
		CpuCapacity: 8,
	})
	suite.NoError(err)
	err = suite.datastore.UpdateUsage("existingCluster2", &central.ClusterMetrics{
		NodeCount:   2,
		CpuCapacity: 7,
	})
	suite.NoError(err)
	u, err = suite.datastore.GetCurrent(context.Background())
	suite.NoError(err)
	suite.Equal(int32(3), u.NumNodes)
	suite.Equal(int32(15), u.NumCores)
	err = suite.datastore.UpdateUsage("unknownCluster", &central.ClusterMetrics{
		NodeCount:   2,
		CpuCapacity: 16,
	})
	suite.NoError(err)
	u, err = suite.datastore.GetCurrent(context.Background())
	suite.NoError(err)
	suite.Equal(int32(3), u.NumNodes)
	suite.Equal(int32(15), u.NumCores)
}

func (suite *UsageDataStoreTestSuite) TestUpdateCutMetrics() {
	u, err := suite.datastore.CutMetrics(context.Background())
	suite.NoError(err)
	suite.Equal(int32(0), u.NumNodes)
	suite.Equal(int32(0), u.NumCores)
	err = suite.datastore.UpdateUsage("existingCluster1", &central.ClusterMetrics{
		NodeCount:   1,
		CpuCapacity: 8,
	})
	suite.NoError(err)
	err = suite.datastore.UpdateUsage("unknownCluster", &central.ClusterMetrics{
		NodeCount:   2,
		CpuCapacity: 7,
	})
	suite.NoError(err)
	u, err = suite.datastore.CutMetrics(context.Background())
	suite.NoError(err)
	suite.Equal(int32(1), u.NumNodes)
	suite.Equal(int32(8), u.NumCores)
	u, err = suite.datastore.CutMetrics(context.Background())
	suite.NoError(err)
	suite.Equal(int32(0), u.NumNodes)
	suite.Equal(int32(0), u.NumCores)
}
