package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/usage/source"
	"github.com/stackrox/rox/central/usage/source/mocks"
	pgMocks "github.com/stackrox/rox/central/usage/store/postgres/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
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
	mockStore *pgMocks.MockStore

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
	suite.mockStore = pgMocks.NewMockStore(suite.ctrl)
	suite.datastore = New(suite.mockStore, &testCluStore{
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

func (suite *UsageDataStoreTestSuite) makeSource(n int64, c int64) source.UsageSource {
	s := mocks.NewMockUsageSource(suite.ctrl)
	s.EXPECT().GetNodeCount().AnyTimes().Return(n)
	s.EXPECT().GetCpuCapacity().AnyTimes().Return(c)
	return s
}

func (suite *UsageDataStoreTestSuite) TestGet() {
	_, err := suite.datastore.Get(suite.hasNoneCtx, nil, nil)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.Get(suite.hasBadCtx, nil, nil)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)

	now := time.Now()
	from := protoconv.ConvertTimeToTimestamp(now.Add(-10 * time.Minute))
	to := protoconv.ConvertTimeToTimestamp(now)
	suite.mockStore.EXPECT().Get(suite.hasReadCtx, from, to).Times(1).Return([]*storage.Usage{
		{Timestamp: from,
			NumNodes:    1,
			NumCpuUnits: 2,
		},
	}, nil)
	u, err := suite.datastore.Get(suite.hasReadCtx, from, to)
	suite.NoError(err)
	suite.Assert().Len(u, 1)
	suite.Equal(int64(1), u[0].GetNumNodes())
	suite.Equal(int64(2), u[0].GetNumCpuUnits())
}

func (suite *UsageDataStoreTestSuite) TestInsert() {
	err := suite.datastore.Insert(suite.hasNoneCtx, &storage.Usage{})
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	err = suite.datastore.Insert(suite.hasBadCtx, &storage.Usage{})
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	err = suite.datastore.Insert(suite.hasReadCtx, &storage.Usage{})
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)

	now := protoconv.ConvertTimeToTimestamp(time.Now())
	metrics := &storage.Usage{
		Timestamp:   now,
		NumNodes:    1,
		NumCpuUnits: 2,
	}
	suite.mockStore.EXPECT().Upsert(suite.hasWriteCtx, metrics).Times(1).Return(nil)
	err = suite.datastore.Insert(suite.hasWriteCtx, metrics)
	suite.NoError(err)
}

func (suite *UsageDataStoreTestSuite) TestGetCurrent() {
	_, err := suite.datastore.GetCurrent(suite.hasNoneCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.GetCurrent(suite.hasBadCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.GetCurrent(suite.hasWriteCtx)
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

func (suite *UsageDataStoreTestSuite) TestCutMetrics() {
	_, err := suite.datastore.CutMetrics(suite.hasNoneCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.CutMetrics(suite.hasBadCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.CutMetrics(suite.hasReadCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (suite *UsageDataStoreTestSuite) TestUpdateGetCurrent() {
	u, err := suite.datastore.GetCurrent(suite.hasReadCtx)
	suite.NoError(err)
	suite.Equal(int64(0), u.NumNodes)
	suite.Equal(int64(0), u.NumCpuUnits)
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "existingCluster1", suite.makeSource(1, 8))
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "existingCluster2", suite.makeSource(2, 7))
	u, err = suite.datastore.GetCurrent(suite.hasReadCtx)
	suite.NoError(err)
	suite.Equal(int64(3), u.NumNodes)
	suite.Equal(int64(15), u.NumCpuUnits)
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "unknownCluster", suite.makeSource(2, 16))
	u, err = suite.datastore.GetCurrent(suite.hasReadCtx)
	suite.NoError(err)
	suite.Equal(int64(3), u.NumNodes)
	suite.Equal(int64(15), u.NumCpuUnits)
}

func (suite *UsageDataStoreTestSuite) TestUpdateCutMetrics() {
	u, err := suite.datastore.CutMetrics(suite.hasWriteCtx)
	suite.NoError(err)
	suite.Equal(int64(0), u.NumNodes)
	suite.Equal(int64(0), u.NumCpuUnits)
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "existingCluster1", suite.makeSource(1, 8))
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "unknownCluster", suite.makeSource(2, 7))
	u, err = suite.datastore.CutMetrics(suite.hasWriteCtx)
	suite.NoError(err)
	suite.Equal(int64(1), u.NumNodes)
	suite.Equal(int64(8), u.NumCpuUnits)
	u, err = suite.datastore.CutMetrics(suite.hasWriteCtx)
	suite.NoError(err)
	suite.Equal(int64(0), u.NumNodes)
	suite.Equal(int64(0), u.NumCpuUnits)
}
