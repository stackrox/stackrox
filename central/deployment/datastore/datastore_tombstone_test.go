//go:build sql_integration

package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestDeploymentDataStoreTombstone(t *testing.T) {
	suite.Run(t, new(DeploymentTombstoneTestSuite))
}

type DeploymentTombstoneTestSuite struct {
	suite.Suite

	testDB              *pgtest.TestPostgres
	ctx                 context.Context
	deploymentDatastore DataStore
}

func (suite *DeploymentTombstoneTestSuite) SetupSuite() {
	suite.ctx = sac.WithAllAccess(context.Background())
	suite.testDB = pgtest.ForT(suite.T())

	var err error
	suite.deploymentDatastore, err = GetTestPostgresDataStore(suite.T(), suite.testDB.DB)
	require.NoError(suite.T(), err)
}

// SetupTest clears the database before each test.
func (suite *DeploymentTombstoneTestSuite) SetupTest() {
	// Clean up any existing deployments.
	deployments, err := suite.deploymentDatastore.GetActiveDeployments(suite.ctx)
	require.NoError(suite.T(), err)
	for _, d := range deployments {
		_ = suite.deploymentDatastore.RemoveDeployment(suite.ctx, d.GetClusterId(), d.GetId())
	}

	deletedDeployments, err := suite.deploymentDatastore.GetSoftDeletedDeployments(suite.ctx)
	require.NoError(suite.T(), err)
	for _, d := range deletedDeployments {
		// Force delete for cleanup (not testing this part).
		_ = suite.deploymentDatastore.RemoveDeployment(suite.ctx, d.GetClusterId(), d.GetId())
	}
}

func (suite *DeploymentTombstoneTestSuite) TestGetActiveDeployments() {
	// Create active deployments.
	activeDeployment1 := fixtures.GetDeployment()
	activeDeployment1.Id = uuid.NewV4().String()
	activeDeployment1.LifecycleStage = storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE

	activeDeployment2 := fixtures.GetDeployment()
	activeDeployment2.Id = uuid.NewV4().String()
	activeDeployment2.LifecycleStage = storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE

	// Create a soft-deleted deployment.
	deletedDeployment := fixtures.GetDeployment()
	deletedDeployment.Id = uuid.NewV4().String()
	deletedDeployment.LifecycleStage = storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED
	deletedDeployment.Tombstone = &storage.Tombstone{
		DeletedAt: timestamppb.Now(),
		ExpiresAt: timestamppb.New(time.Now().Add(24 * time.Hour)),
	}

	// Upsert all deployments.
	require.NoError(suite.T(), suite.deploymentDatastore.UpsertDeployment(suite.ctx, activeDeployment1))
	require.NoError(suite.T(), suite.deploymentDatastore.UpsertDeployment(suite.ctx, activeDeployment2))
	require.NoError(suite.T(), suite.deploymentDatastore.UpsertDeployment(suite.ctx, deletedDeployment))

	// Get active deployments.
	activeDeployments, err := suite.deploymentDatastore.GetActiveDeployments(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify only active deployments are returned.
	require.Len(suite.T(), activeDeployments, 2)
	activeIDs := make(map[string]bool)
	for _, d := range activeDeployments {
		activeIDs[d.GetId()] = true
		assert.Equal(suite.T(), storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE, d.GetLifecycleStage())
	}
	assert.True(suite.T(), activeIDs[activeDeployment1.GetId()])
	assert.True(suite.T(), activeIDs[activeDeployment2.GetId()])
	assert.False(suite.T(), activeIDs[deletedDeployment.GetId()])
}

func (suite *DeploymentTombstoneTestSuite) TestGetSoftDeletedDeployments() {
	// Create active deployment.
	activeDeployment := fixtures.GetDeployment()
	activeDeployment.Id = uuid.NewV4().String()
	activeDeployment.LifecycleStage = storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE

	// Create soft-deleted deployments.
	deletedDeployment1 := fixtures.GetDeployment()
	deletedDeployment1.Id = uuid.NewV4().String()
	deletedDeployment1.LifecycleStage = storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED
	deletedDeployment1.Tombstone = &storage.Tombstone{
		DeletedAt: timestamppb.Now(),
		ExpiresAt: timestamppb.New(time.Now().Add(24 * time.Hour)),
	}

	deletedDeployment2 := fixtures.GetDeployment()
	deletedDeployment2.Id = uuid.NewV4().String()
	deletedDeployment2.LifecycleStage = storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED
	deletedDeployment2.Tombstone = &storage.Tombstone{
		DeletedAt: timestamppb.Now(),
		ExpiresAt: timestamppb.New(time.Now().Add(48 * time.Hour)),
	}

	// Upsert all deployments.
	require.NoError(suite.T(), suite.deploymentDatastore.UpsertDeployment(suite.ctx, activeDeployment))
	require.NoError(suite.T(), suite.deploymentDatastore.UpsertDeployment(suite.ctx, deletedDeployment1))
	require.NoError(suite.T(), suite.deploymentDatastore.UpsertDeployment(suite.ctx, deletedDeployment2))

	// Get soft-deleted deployments.
	deletedDeployments, err := suite.deploymentDatastore.GetSoftDeletedDeployments(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify only soft-deleted deployments are returned.
	require.Len(suite.T(), deletedDeployments, 2)
	deletedIDs := make(map[string]bool)
	for _, d := range deletedDeployments {
		deletedIDs[d.GetId()] = true
		assert.Equal(suite.T(), storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED, d.GetLifecycleStage())
		assert.NotNil(suite.T(), d.GetTombstone())
	}
	assert.True(suite.T(), deletedIDs[deletedDeployment1.GetId()])
	assert.True(suite.T(), deletedIDs[deletedDeployment2.GetId()])
	assert.False(suite.T(), deletedIDs[activeDeployment.GetId()])
}

func (suite *DeploymentTombstoneTestSuite) TestGetExpiredDeployments() {
	now := time.Now()

	// Create active deployment (should not be returned).
	activeDeployment := fixtures.GetDeployment()
	activeDeployment.Id = uuid.NewV4().String()
	activeDeployment.LifecycleStage = storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE

	// Create expired soft-deleted deployment (expires in the past).
	expiredDeployment1 := fixtures.GetDeployment()
	expiredDeployment1.Id = uuid.NewV4().String()
	expiredDeployment1.LifecycleStage = storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED
	expiredDeployment1.Tombstone = &storage.Tombstone{
		DeletedAt: timestamppb.New(now.Add(-48 * time.Hour)),
		ExpiresAt: timestamppb.New(now.Add(-24 * time.Hour)), // Expired 24 hours ago.
	}

	expiredDeployment2 := fixtures.GetDeployment()
	expiredDeployment2.Id = uuid.NewV4().String()
	expiredDeployment2.LifecycleStage = storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED
	expiredDeployment2.Tombstone = &storage.Tombstone{
		DeletedAt: timestamppb.New(now.Add(-2 * time.Hour)),
		ExpiresAt: timestamppb.New(now.Add(-1 * time.Hour)), // Expired 1 hour ago.
	}

	// Create non-expired soft-deleted deployment (expires in the future).
	notExpiredDeployment := fixtures.GetDeployment()
	notExpiredDeployment.Id = uuid.NewV4().String()
	notExpiredDeployment.LifecycleStage = storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED
	notExpiredDeployment.Tombstone = &storage.Tombstone{
		DeletedAt: timestamppb.Now(),
		ExpiresAt: timestamppb.New(now.Add(24 * time.Hour)), // Expires in 24 hours.
	}

	// Upsert all deployments.
	require.NoError(suite.T(), suite.deploymentDatastore.UpsertDeployment(suite.ctx, activeDeployment))
	require.NoError(suite.T(), suite.deploymentDatastore.UpsertDeployment(suite.ctx, expiredDeployment1))
	require.NoError(suite.T(), suite.deploymentDatastore.UpsertDeployment(suite.ctx, expiredDeployment2))
	require.NoError(suite.T(), suite.deploymentDatastore.UpsertDeployment(suite.ctx, notExpiredDeployment))

	// Get expired deployments.
	expiredDeployments, err := suite.deploymentDatastore.GetExpiredDeployments(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify only expired deployments are returned.
	require.Len(suite.T(), expiredDeployments, 2)
	expiredIDs := make(map[string]bool)
	for _, d := range expiredDeployments {
		expiredIDs[d.GetId()] = true
		assert.Equal(suite.T(), storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED, d.GetLifecycleStage())
		assert.NotNil(suite.T(), d.GetTombstone())
		expiresAt := d.GetTombstone().GetExpiresAt().AsTime()
		assert.True(suite.T(), expiresAt.Before(now), "Deployment %s should be expired (expires_at=%v, now=%v)", d.GetId(), expiresAt, now)
	}
	assert.True(suite.T(), expiredIDs[expiredDeployment1.GetId()])
	assert.True(suite.T(), expiredIDs[expiredDeployment2.GetId()])
	assert.False(suite.T(), expiredIDs[notExpiredDeployment.GetId()])
	assert.False(suite.T(), expiredIDs[activeDeployment.GetId()])
}

func (suite *DeploymentTombstoneTestSuite) TestGetExpiredDeployments_NoTombstone() {
	// Create soft-deleted deployment without tombstone (should not crash).
	deletedDeploymentNoTombstone := fixtures.GetDeployment()
	deletedDeploymentNoTombstone.Id = uuid.NewV4().String()
	deletedDeploymentNoTombstone.LifecycleStage = storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED
	deletedDeploymentNoTombstone.Tombstone = nil

	require.NoError(suite.T(), suite.deploymentDatastore.UpsertDeployment(suite.ctx, deletedDeploymentNoTombstone))

	// Get expired deployments (should not crash or include deployment without tombstone).
	expiredDeployments, err := suite.deploymentDatastore.GetExpiredDeployments(suite.ctx)
	require.NoError(suite.T(), err)

	// Should not include deployment without tombstone.
	for _, d := range expiredDeployments {
		assert.NotEqual(suite.T(), deletedDeploymentNoTombstone.GetId(), d.GetId())
	}
}

func (suite *DeploymentTombstoneTestSuite) TestGetExpiredDeployments_ExactlyNow() {
	now := protocompat.TimestampNow()

	// Create soft-deleted deployment that expires exactly now.
	// The implementation uses Before(), so this should NOT be included.
	expiresNowDeployment := fixtures.GetDeployment()
	expiresNowDeployment.Id = uuid.NewV4().String()
	expiresNowDeployment.LifecycleStage = storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED
	expiresNowDeployment.Tombstone = &storage.Tombstone{
		DeletedAt: timestamppb.New(now.AsTime().Add(-1 * time.Hour)),
		ExpiresAt: now,
	}

	require.NoError(suite.T(), suite.deploymentDatastore.UpsertDeployment(suite.ctx, expiresNowDeployment))

	// Get expired deployments.
	expiredDeployments, err := suite.deploymentDatastore.GetExpiredDeployments(suite.ctx)
	require.NoError(suite.T(), err)

	// Should not include deployment that expires exactly now (using Before, not BeforeOrEqual).
	for _, d := range expiredDeployments {
		assert.NotEqual(suite.T(), expiresNowDeployment.GetId(), d.GetId())
	}
}
