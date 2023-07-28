package datastore

import (
	"context"
	"testing"

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

type testMetricsSource [2]int64

func (tms *testMetricsSource) GetNodeCount() int64   { return tms[0] }
func (tms *testMetricsSource) GetCpuCapacity() int64 { return tms[1] }

func (suite *UsageDataStoreTestSuite) TestUpdateGetCurrent() {
	u, err := suite.datastore.GetCurrent(context.Background())
	suite.NoError(err)
	suite.Equal(int32(0), u.NumNodes)
	suite.Equal(int32(0), u.NumCpuUnits)
	suite.datastore.UpdateUsage("existingCluster1", &testMetricsSource{1, 8})
	suite.datastore.UpdateUsage("existingCluster2", &testMetricsSource{2, 7})
	u, err = suite.datastore.GetCurrent(context.Background())
	suite.NoError(err)
	suite.Equal(int32(3), u.NumNodes)
	suite.Equal(int32(15), u.NumCpuUnits)
	suite.datastore.UpdateUsage("unknownCluster", &testMetricsSource{2, 16})
	u, err = suite.datastore.GetCurrent(context.Background())
	suite.NoError(err)
	suite.Equal(int32(3), u.NumNodes)
	suite.Equal(int32(15), u.NumCpuUnits)
}

func (suite *UsageDataStoreTestSuite) TestUpdateCutMetrics() {
	u, err := suite.datastore.CutMetrics(context.Background())
	suite.NoError(err)
	suite.Equal(int32(0), u.NumNodes)
	suite.Equal(int32(0), u.NumCpuUnits)
	suite.datastore.UpdateUsage("existingCluster1", &testMetricsSource{1, 8})
	suite.datastore.UpdateUsage("unknownCluster", &testMetricsSource{2, 7})
	u, err = suite.datastore.CutMetrics(context.Background())
	suite.NoError(err)
	suite.Equal(int32(1), u.NumNodes)
	suite.Equal(int32(8), u.NumCpuUnits)
	u, err = suite.datastore.CutMetrics(context.Background())
	suite.NoError(err)
	suite.Equal(int32(0), u.NumNodes)
	suite.Equal(int32(0), u.NumCpuUnits)
}
