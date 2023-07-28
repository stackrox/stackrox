package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/usage/source"
	"github.com/stackrox/rox/central/usage/source/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestUsageDataStore(t *testing.T) {
	suite.Run(t, new(UsageDataStoreTestSuite))
}

type UsageDataStoreTestSuite struct {
	suite.Suite

	datastore DataStore
	ctrl      *gomock.Controller
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
	suite.ctrl = gomock.NewController(suite.T())
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

func (suite *UsageDataStoreTestSuite) makeSource(n int64, c int64) source.UsageSource {
	s := mocks.NewMockUsageSource(suite.ctrl)
	s.EXPECT().GetNodeCount().AnyTimes().Return(n)
	s.EXPECT().GetCpuCapacity().AnyTimes().Return(c)
	return s
}

func (suite *UsageDataStoreTestSuite) TestUpdateGetCurrent() {
	u, err := suite.datastore.GetCurrent(context.Background())
	suite.NoError(err)
	suite.Equal(int64(0), u.NumNodes)
	suite.Equal(int64(0), u.NumCpuUnits)
	suite.datastore.UpdateUsage("existingCluster1", suite.makeSource(1, 8))
	suite.datastore.UpdateUsage("existingCluster2", suite.makeSource(2, 7))
	u, err = suite.datastore.GetCurrent(context.Background())
	suite.NoError(err)
	suite.Equal(int64(3), u.NumNodes)
	suite.Equal(int64(15), u.NumCpuUnits)
	suite.datastore.UpdateUsage("unknownCluster", suite.makeSource(2, 16))
	u, err = suite.datastore.GetCurrent(context.Background())
	suite.NoError(err)
	suite.Equal(int64(3), u.NumNodes)
	suite.Equal(int64(15), u.NumCpuUnits)
}

func (suite *UsageDataStoreTestSuite) TestUpdateCutMetrics() {
	u, err := suite.datastore.CutMetrics(context.Background())
	suite.NoError(err)
	suite.Equal(int64(0), u.NumNodes)
	suite.Equal(int64(0), u.NumCpuUnits)
	suite.datastore.UpdateUsage("existingCluster1", suite.makeSource(1, 8))
	suite.datastore.UpdateUsage("unknownCluster", suite.makeSource(2, 7))
	u, err = suite.datastore.CutMetrics(context.Background())
	suite.NoError(err)
	suite.Equal(int64(1), u.NumNodes)
	suite.Equal(int64(8), u.NumCpuUnits)
	u, err = suite.datastore.CutMetrics(context.Background())
	suite.NoError(err)
	suite.Equal(int64(0), u.NumNodes)
	suite.Equal(int64(0), u.NumCpuUnits)
}
