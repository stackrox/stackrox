//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/networkbaseline/store"
	pgStore "github.com/stackrox/rox/central/networkbaseline/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

var (
	allAllowedCtx    = sac.WithAllAccess(context.Background())
	expectedBaseline = fixtures.GetNetworkBaseline()
)

func TestNetworkBaselineDatastoreSuite(t *testing.T) {
	suite.Run(t, new(NetworkBaselineDataStoreTestSuite))
}

type NetworkBaselineDataStoreTestSuite struct {
	suite.Suite

	datastore DataStore
	storage   store.Store
	pool      postgres.DB
}

var _ interface {
	suite.SetupAllSuite
	suite.TearDownAllSuite
} = (*NetworkBaselineDataStoreTestSuite)(nil)

func (suite *NetworkBaselineDataStoreTestSuite) SetupSuite() {
	ctx := context.Background()
	source := pgtest.GetConnectionString(suite.T())
	config, err := postgres.ParseConfig(source)
	suite.NoError(err)
	suite.pool, err = postgres.New(ctx, config)
	suite.NoError(err)
	pgStore.Destroy(ctx, suite.pool)
	gormDB := pgtest.OpenGormDB(suite.T(), source)
	defer pgtest.CloseGormDB(suite.T(), gormDB)
	suite.storage = pgStore.CreateTableAndNewStore(ctx, suite.pool, gormDB)
	suite.datastore = newNetworkBaselineDataStore(suite.storage)
}

func (suite *NetworkBaselineDataStoreTestSuite) TearDownSuite() {
	suite.pool.Close()
}

func (suite *NetworkBaselineDataStoreTestSuite) mustGetBaseline(ctx context.Context, deploymentID string) (*storage.NetworkBaseline, bool) {
	baseline, found, err := suite.datastore.GetNetworkBaseline(ctx, deploymentID)
	suite.Require().NoError(err)
	return baseline, found
}

func (suite *NetworkBaselineDataStoreTestSuite) TestNoAccessAllowed() {
	// First create a baseline in datastore to make sure when we return false on get
	// we are indeed hitting permission issue
	suite.Nil(suite.datastore.UpsertNetworkBaselines(allAllowedCtx, []*storage.NetworkBaseline{expectedBaseline}))

	ctx := sac.WithNoAccess(context.Background())

	_, ok := suite.mustGetBaseline(ctx, expectedBaseline.GetDeploymentId())
	suite.False(ok)

	suite.Error(suite.datastore.UpsertNetworkBaselines(ctx, []*storage.NetworkBaseline{expectedBaseline}), "permission denied")

	suite.Error(suite.datastore.DeleteNetworkBaseline(ctx, expectedBaseline.GetDeploymentId()), "permission denied")
	// BTW if we try to delete non-existent/already deleted baseline, it should just return nil
	suite.Nil(suite.datastore.DeleteNetworkBaseline(ctx, uuid.Nil.String()))
}

func (suite *NetworkBaselineDataStoreTestSuite) TestNetworkBaselines() {
	// With all allowed access, we should be able to perform all ops on datastore
	// Create
	suite.Nil(suite.datastore.UpsertNetworkBaselines(allAllowedCtx, []*storage.NetworkBaseline{expectedBaseline}))

	baseline, ok := suite.mustGetBaseline(allAllowedCtx, expectedBaseline.GetDeploymentId())
	suite.True(ok)
	suite.Equal(expectedBaseline.GetClusterId(), baseline.GetClusterId())
	suite.Equal(expectedBaseline.GetNamespace(), baseline.GetNamespace())
	suite.Equal(expectedBaseline.GetLocked(), baseline.GetLocked())

	// Update
	originalBaselineLocked := expectedBaseline.GetLocked()
	expectedBaseline.Locked = !expectedBaseline.GetLocked()
	suite.Nil(suite.datastore.UpsertNetworkBaselines(allAllowedCtx, []*storage.NetworkBaseline{expectedBaseline}))
	// Check update
	baseline, ok = suite.mustGetBaseline(allAllowedCtx, expectedBaseline.GetDeploymentId())
	suite.True(ok)
	suite.NotEqual(originalBaselineLocked, baseline.GetLocked())
	suite.Equal(baseline.GetLocked(), expectedBaseline.GetLocked())

	// Delete
	suite.Nil(suite.datastore.DeleteNetworkBaseline(allAllowedCtx, expectedBaseline.GetDeploymentId()))

	// Verify deletion
	_, ok = suite.mustGetBaseline(allAllowedCtx, expectedBaseline.GetDeploymentId())
	suite.False(ok)
}

func (suite *NetworkBaselineDataStoreTestSuite) TestSAC() {
	ctxWithWrongClusterReadAccess :=
		sac.WithGlobalAccessScopeChecker(
			context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.DeploymentExtension),
				sac.ClusterScopeKeys(testconsts.Cluster3)))

	ctxWithReadAccess :=
		sac.WithGlobalAccessScopeChecker(
			context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.DeploymentExtension),
				sac.ClusterScopeKeys(expectedBaseline.GetClusterId()),
				sac.NamespaceScopeKeys(expectedBaseline.GetNamespace())))

	ctxWithWrongClusterWriteAccess :=
		sac.WithGlobalAccessScopeChecker(
			context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.DeploymentExtension),
				sac.ClusterScopeKeys(testconsts.Cluster3)))

	ctxWithWriteAccess :=
		sac.WithGlobalAccessScopeChecker(
			context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.DeploymentExtension),
				sac.ClusterScopeKeys(expectedBaseline.GetClusterId()),
				sac.NamespaceScopeKeys(expectedBaseline.GetNamespace())))

	ctxWithUnrestrictedWriteAccess :=
		sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.DeploymentExtension)))

	// Test Update
	{
		expectedBaseline.Locked = !expectedBaseline.Locked
		suite.Error(suite.datastore.UpsertNetworkBaselines(ctxWithWrongClusterReadAccess, []*storage.NetworkBaseline{expectedBaseline}), "permission denied")
		suite.Error(suite.datastore.UpsertNetworkBaselines(ctxWithReadAccess, []*storage.NetworkBaseline{expectedBaseline}), "permission denied")
		suite.Error(suite.datastore.UpsertNetworkBaselines(ctxWithWrongClusterWriteAccess, []*storage.NetworkBaseline{expectedBaseline}), "permission denied")
		suite.Nil(suite.datastore.UpsertNetworkBaselines(ctxWithWriteAccess, []*storage.NetworkBaseline{expectedBaseline}))
		// Check updated value
		result, found := suite.mustGetBaseline(allAllowedCtx, expectedBaseline.GetDeploymentId())
		suite.True(found)
		suite.Equal(expectedBaseline.Locked, result.Locked)
	}

	// Test Get
	{
		_, ok := suite.mustGetBaseline(ctxWithWrongClusterReadAccess, expectedBaseline.GetDeploymentId())
		suite.False(ok)
		_, ok = suite.mustGetBaseline(ctxWithReadAccess, expectedBaseline.GetDeploymentId())
		suite.True(ok)
		_, ok = suite.mustGetBaseline(ctxWithWrongClusterWriteAccess, expectedBaseline.GetDeploymentId())
		suite.False(ok)
		_, ok = suite.mustGetBaseline(ctxWithWriteAccess, expectedBaseline.GetDeploymentId())
		suite.False(ok)
	}

	// Test Delete
	{
		suite.Error(suite.datastore.DeleteNetworkBaseline(ctxWithWrongClusterReadAccess, expectedBaseline.GetDeploymentId()), "permission denied")
		suite.Error(suite.datastore.DeleteNetworkBaseline(ctxWithReadAccess, expectedBaseline.GetDeploymentId()), "permission denied")
		suite.Error(suite.datastore.DeleteNetworkBaseline(ctxWithWrongClusterWriteAccess, expectedBaseline.GetDeploymentId()), "permission denied")
		suite.Nil(suite.datastore.DeleteNetworkBaseline(ctxWithWriteAccess, expectedBaseline.GetDeploymentId()))
		_, ok := suite.mustGetBaseline(ctxWithReadAccess, expectedBaseline.GetDeploymentId())
		suite.False(ok)
	}

	// Test Delete Multiple
	{
		nbs := fixtures.GetSACTestNetworkBaseline()
		var ids []string
		for _, nb := range nbs {
			ids = append(ids, nb.GetDeploymentId())
		}

		suite.Nil(suite.datastore.UpsertNetworkBaselines(ctxWithUnrestrictedWriteAccess, nbs))
		suite.Error(suite.datastore.DeleteNetworkBaselines(ctxWithWrongClusterReadAccess, ids), "permission denied")
		suite.Error(suite.datastore.DeleteNetworkBaselines(ctxWithReadAccess, ids), "permission denied")
		suite.Error(suite.datastore.DeleteNetworkBaselines(ctxWithWrongClusterWriteAccess, ids), "permission denied")
		suite.Nil(suite.datastore.DeleteNetworkBaselines(ctxWithUnrestrictedWriteAccess, ids))
		for _, id := range ids {
			_, ok := suite.mustGetBaseline(ctxWithReadAccess, id)
			suite.False(ok)
		}
	}
}
