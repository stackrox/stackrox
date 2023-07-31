package datastore

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	mockStore "github.com/stackrox/rox/central/productusage/store/mocks"
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
	store     *mockStore.MockStore
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
	suite.store = mockStore.NewMockStore(suite.ctrl)
	suite.datastore = New(suite.store, &testCluStore{
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

func makeSource(n int64, c int64) *storage.SecuredUnits {
	return &storage.SecuredUnits{
		NumNodes:    n,
		NumCpuUnits: c,
	}
}

func drain[T any](ch <-chan T) {
	for range ch {
		<-ch
	}
}

func (suite *UsageDataStoreTestSuite) TestGet() {
	_, err := suite.datastore.Get(suite.hasNoneCtx, nil, nil)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.Get(suite.hasBadCtx, nil, nil)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)

	suite.store.EXPECT().Walk(gomock.Any(), gomock.Any()).Times(4).Return(nil)
	ch, err := suite.datastore.Get(suite.hasReadCtx, nil, nil)
	suite.NoError(err)
	drain(ch)
	ch, err = suite.datastore.Get(suite.hasReadCtx, &types.Timestamp{}, nil)
	suite.NoError(err)
	drain(ch)
	ch, err = suite.datastore.Get(suite.hasReadCtx, nil, &types.Timestamp{})
	suite.NoError(err)
	drain(ch)
	ch, err = suite.datastore.Get(suite.hasWriteCtx, &types.Timestamp{}, &types.Timestamp{})
	suite.NoError(err)
	drain(ch)
}

func (suite *UsageDataStoreTestSuite) TestInsert() {
	err := suite.datastore.Insert(suite.hasNoneCtx, &storage.SecuredUnits{})
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	err = suite.datastore.Insert(suite.hasBadCtx, &storage.SecuredUnits{})
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	err = suite.datastore.Insert(suite.hasReadCtx, &storage.SecuredUnits{})
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)

	suite.store.EXPECT().Upsert(suite.hasWriteCtx, gomock.Any()).Times(1).Return(nil)
	err = suite.datastore.Insert(suite.hasWriteCtx, &storage.SecuredUnits{})
	suite.NoError(err)
}

func (suite *UsageDataStoreTestSuite) TestGetCurrentPermissions() {
	_, err := suite.datastore.GetCurrentUsage(suite.hasNoneCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.GetCurrentUsage(suite.hasBadCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.GetCurrentUsage(suite.hasWriteCtx)
	suite.NoError(err)
}

func (suite *UsageDataStoreTestSuite) TestUpdateUsagePermissions() {
	err := suite.datastore.UpdateUsage(suite.hasNoneCtx, "existingCluster1", makeSource(1, 8))
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	err = suite.datastore.UpdateUsage(suite.hasBadCtx, "existingCluster1", makeSource(1, 8))
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	err = suite.datastore.UpdateUsage(suite.hasReadCtx, "existingCluster1", makeSource(1, 8))
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (suite *UsageDataStoreTestSuite) TestAggregateAndFlushPermissions() {
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
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "existingCluster1", makeSource(1, 8))
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "existingCluster2", makeSource(2, 7))
	u, err = suite.datastore.GetCurrentUsage(suite.hasReadCtx)
	suite.NoError(err)
	suite.Equal(int64(3), u.NumNodes)
	suite.Equal(int64(15), u.NumCpuUnits)
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "unknownCluster", makeSource(2, 16))
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
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "existingCluster1", makeSource(1, 8))
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "unknownCluster", makeSource(2, 7))
	u, err = suite.datastore.AggregateAndFlush(suite.hasWriteCtx)
	suite.NoError(err)
	suite.Equal(int64(1), u.NumNodes)
	suite.Equal(int64(8), u.NumCpuUnits)
	u, err = suite.datastore.AggregateAndFlush(suite.hasWriteCtx)
	suite.NoError(err)
	suite.Equal(int64(0), u.NumNodes)
	suite.Equal(int64(0), u.NumCpuUnits)
}
