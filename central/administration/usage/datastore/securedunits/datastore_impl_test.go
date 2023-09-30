//go:build sql_integration

package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/administration/usage/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
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
	store     *pgtest.TestPostgres
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

	suite.store = pgtest.ForT(suite.T())
	suite.Require().NotNil(suite.store)
	store := postgres.New(suite.store)

	suite.datastore = New(store, &testCluStore{
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

func (suite *UsageDataStoreTestSuite) TearDownTest() {
	suite.store.Close()
}

func makeSource(n int64, c int64) *storage.SecuredUnits {
	return &storage.SecuredUnits{
		NumNodes:    n,
		NumCpuUnits: c,
	}
}

func (suite *UsageDataStoreTestSuite) TestWalk() {
	var zeroTime time.Time
	err := suite.datastore.Walk(suite.hasNoneCtx, zeroTime, zeroTime, nil)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	err = suite.datastore.Walk(suite.hasBadCtx, zeroTime, zeroTime, nil)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)

	const N = page + 2
	first := time.Now()
	last := first
	for i := 0; i < N; i++ {
		ts, _ := types.TimestampProto(last)
		err = suite.datastore.Add(suite.hasWriteCtx, &storage.SecuredUnits{
			Timestamp:   ts,
			NumNodes:    int64(i),
			NumCpuUnits: int64(i * 2),
		})
		suite.Require().NoError(err)
		last = last.Add(5 * time.Minute)
	}

	var totalNodes, totalCPUUnits int64
	fn := func(su *storage.SecuredUnits) error {
		totalNodes += su.NumNodes
		totalCPUUnits += su.NumCpuUnits
		return nil
	}

	err = suite.datastore.Walk(suite.hasReadCtx, first, last, fn)
	suite.NoError(err)
	suite.Equal(int64((N-1)*N/2), totalNodes)
	suite.Equal(int64((N-1)*N), totalCPUUnits)
}

func (suite *UsageDataStoreTestSuite) TestGetMax() {
	var zeroTime time.Time
	_, err := suite.datastore.GetMaxNumNodes(suite.hasNoneCtx, zeroTime, zeroTime)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.GetMaxNumNodes(suite.hasBadCtx, zeroTime, zeroTime)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)

	const N = page + 2
	first := time.Now()
	last := first
	for i := 0; i < N; i++ {
		ts, _ := types.TimestampProto(last)
		err = suite.datastore.Add(suite.hasWriteCtx, &storage.SecuredUnits{
			Timestamp:   ts,
			NumNodes:    int64(i),
			NumCpuUnits: int64(i * 2),
		})
		suite.Require().NoError(err)
		last = last.Add(5 * time.Minute)
	}

	// Overall maximum (last entry):
	units, err := suite.datastore.GetMaxNumNodes(suite.hasReadCtx, first, last)
	suite.NoError(err)
	suite.Equal(last.Add(-5*time.Minute).Unix(), units.Timestamp.Seconds)
	suite.Equal(int64(N-1), units.NumNodes)
	suite.Equal(int64((N-1)*2), units.NumCpuUnits)

	// Maximum in a range:
	last = first.Add(3 * 5 * time.Minute)
	units, err = suite.datastore.GetMaxNumCPUUnits(suite.hasReadCtx, first, last)
	suite.NoError(err)
	suite.Equal(last.Add(-5*time.Minute).Unix(), units.Timestamp.Seconds)
	suite.Equal(int64(3-1), units.NumNodes)
	suite.Equal(int64((3-1)*2), units.NumCpuUnits)
}

func (suite *UsageDataStoreTestSuite) TestAdd() {
	err := suite.datastore.Add(suite.hasNoneCtx, &storage.SecuredUnits{})
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	err = suite.datastore.Add(suite.hasBadCtx, &storage.SecuredUnits{})
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	err = suite.datastore.Add(suite.hasReadCtx, &storage.SecuredUnits{})
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)

	err = suite.datastore.Add(suite.hasWriteCtx, &storage.SecuredUnits{})
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

func (suite *UsageDataStoreTestSuite) TestAggregateAndResetPermissions() {
	_, err := suite.datastore.AggregateAndReset(suite.hasNoneCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.AggregateAndReset(suite.hasBadCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, err = suite.datastore.AggregateAndReset(suite.hasReadCtx)
	suite.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (suite *UsageDataStoreTestSuite) TestUpdateGetCurrent() {
	u, err := suite.datastore.GetCurrentUsage(suite.hasReadCtx)
	suite.NoError(err)
	suite.Equal(int64(0), u.GetNumNodes())
	suite.Equal(int64(0), u.GetNumCpuUnits())
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "existingCluster1", makeSource(1, 8))
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "existingCluster2", makeSource(2, 7))
	u, err = suite.datastore.GetCurrentUsage(suite.hasReadCtx)
	suite.NoError(err)
	suite.Equal(int64(3), u.GetNumNodes())
	suite.Equal(int64(15), u.GetNumCpuUnits())
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "unknownCluster", makeSource(2, 16))
	u, err = suite.datastore.GetCurrentUsage(suite.hasReadCtx)
	suite.NoError(err)
	suite.Equal(int64(3), u.GetNumNodes())
	suite.Equal(int64(15), u.GetNumCpuUnits())
}

func (suite *UsageDataStoreTestSuite) TestUpdateAggregateAndReset() {
	u, err := suite.datastore.AggregateAndReset(suite.hasWriteCtx)
	suite.NoError(err)
	suite.Equal(int64(0), u.GetNumNodes())
	suite.Equal(int64(0), u.GetNumCpuUnits())
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "existingCluster1", makeSource(1, 8))
	_ = suite.datastore.UpdateUsage(suite.hasWriteCtx, "unknownCluster", makeSource(2, 7))
	u, err = suite.datastore.AggregateAndReset(suite.hasWriteCtx)
	suite.NoError(err)
	suite.Equal(int64(1), u.GetNumNodes())
	suite.Equal(int64(8), u.GetNumCpuUnits())
	u, err = suite.datastore.AggregateAndReset(suite.hasWriteCtx)
	suite.NoError(err)
	suite.Equal(int64(0), u.GetNumNodes())
	suite.Equal(int64(0), u.GetNumCpuUnits())
}
