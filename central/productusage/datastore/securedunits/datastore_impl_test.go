package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/productusage/source"
	"github.com/stackrox/rox/central/productusage/source/mocks"
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

	hasNoneCtx  context.Context
	hasBadCtx   context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context
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
	suite.datastore = New(&testCluStore{
		clusters: []*storage.Cluster{{
			Id: "existingCluster1",
		}, {
			Id: "existingCluster2",
		}},
	})

	suite.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	suite.hasBadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert)))
	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
}

func (suite *UsageDataStoreTestSuite) TearDownSuite() {
}

func (suite *UsageDataStoreTestSuite) makeSource(n int64, c int64) source.SecuredUnitsSource {
	s := mocks.NewMockSecuredUnitsSource(suite.ctrl)
	s.EXPECT().GetNodeCount().AnyTimes().Return(n)
	s.EXPECT().GetCpuCapacity().AnyTimes().Return(c)
	return s
}

func (suite *UsageDataStoreTestSuite) TestGet() {
	_, err := suite.datastore.Get(suite.hasNoneCtx, nil, nil)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.Get(suite.hasBadCtx, nil, nil)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (suite *UsageDataStoreTestSuite) TestInsert() {
	err := suite.datastore.Insert(suite.hasNoneCtx, &storage.SecuredUnits{})
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	err = suite.datastore.Insert(suite.hasBadCtx, &storage.SecuredUnits{})
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	err = suite.datastore.Insert(suite.hasReadCtx, &storage.SecuredUnits{})
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (suite *UsageDataStoreTestSuite) TestGetCurrent() {
	_, err := suite.datastore.GetCurrentUsage(suite.hasNoneCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.GetCurrentUsage(suite.hasBadCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.GetCurrentUsage(suite.hasWriteCtx)
	suite.NoError(err)
}

func (suite *UsageDataStoreTestSuite) TestUpdateUsage() {
	err := suite.datastore.UpdateUsage(suite.hasNoneCtx, "existingCluster1", suite.makeSource(1, 8))
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	err = suite.datastore.UpdateUsage(suite.hasBadCtx, "existingCluster1", suite.makeSource(1, 8))
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	err = suite.datastore.UpdateUsage(suite.hasReadCtx, "existingCluster1", suite.makeSource(1, 8))
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (suite *UsageDataStoreTestSuite) TestAggregateAndFlush() {
	_, err := suite.datastore.AggregateAndFlush(suite.hasNoneCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.AggregateAndFlush(suite.hasBadCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.AggregateAndFlush(suite.hasReadCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (suite *UsageDataStoreTestSuite) TestUpdateGetCurrent() {
	u, err := suite.datastore.GetCurrentUsage(suite.hasReadCtx)
	suite.NoError(err)
	suite.Equal(int64(0), u.NumNodes)
	suite.Equal(int64(0), u.NumCpuUnits)
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "existingCluster1", suite.makeSource(1, 8))
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "existingCluster2", suite.makeSource(2, 7))
	u, err = suite.datastore.GetCurrentUsage(suite.hasReadCtx)
	suite.NoError(err)
	suite.Equal(int64(3), u.NumNodes)
	suite.Equal(int64(15), u.NumCpuUnits)
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "unknownCluster", suite.makeSource(2, 16))
	u, err = suite.datastore.GetCurrentUsage(suite.hasReadCtx)
	suite.NoError(err)
	suite.Equal(int64(3), u.NumNodes)
	suite.Equal(int64(15), u.NumCpuUnits)
}

func (suite *UsageDataStoreTestSuite) TestUpdateAggregateAndFlush() {
	u, err := suite.datastore.AggregateAndFlush(suite.hasWriteCtx)
	suite.NoError(err)
	suite.Equal(int64(0), u.NumNodes)
	suite.Equal(int64(0), u.NumCpuUnits)
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "existingCluster1", suite.makeSource(1, 8))
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "unknownCluster", suite.makeSource(2, 7))
	u, err = suite.datastore.AggregateAndFlush(suite.hasWriteCtx)
	suite.NoError(err)
	suite.Equal(int64(1), u.NumNodes)
	suite.Equal(int64(8), u.NumCpuUnits)
	u, err = suite.datastore.AggregateAndFlush(suite.hasWriteCtx)
	suite.NoError(err)
	suite.Equal(int64(0), u.NumNodes)
	suite.Equal(int64(0), u.NumCpuUnits)
}
