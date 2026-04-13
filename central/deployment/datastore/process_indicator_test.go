//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	processFilterMocks "github.com/stackrox/rox/pkg/process/filter/mocks"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestProcessIndicatorCleanupOnSoftDelete verifies that process indicators are properly
// cleaned up when a deployment is soft-deleted.
// This addresses design review comment #7: Verify the process indicator queue still removes
// the deployment when it is deleted.
func TestProcessIndicatorCleanupOnSoftDelete(t *testing.T) {
	testDB := pgtest.ForT(t)
	defer testDB.Close()

	ctx := sac.WithAllAccess(context.Background())

	// Create a mock process filter to verify Delete is called.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockProcessFilter := processFilterMocks.NewMockFilter(ctrl)

	// Create the datastore with the mock process filter.
	ds, err := GetTestPostgresDataStoreWithProcessFilter(t, testDB.DB, mockProcessFilter)
	require.NoError(t, err)

	// Create a test deployment.
	deployment := fixtures.GetDeployment()
	deployment.Id = uuid.NewV4().String()
	deployment.Name = "test-deployment-with-processes"
	deployment.ClusterId = "test-cluster"
	deployment.LifecycleStage = storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE

	// Upsert the deployment.
	require.NoError(t, ds.UpsertDeployment(ctx, deployment))

	// Verify deployment exists.
	retrieved, exists, err := ds.GetDeployment(ctx, deployment.GetId())
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE, retrieved.GetLifecycleStage())

	// Expect the process filter's Delete method to be called when we soft-delete the deployment.
	mockProcessFilter.EXPECT().Delete(deployment.GetId()).Times(1)

	// Soft-delete the deployment.
	err = ds.RemoveDeployment(ctx, deployment.GetClusterId(), deployment.GetId())
	require.NoError(t, err)

	// Verify the deployment is now soft-deleted.
	deleted, exists, err := ds.GetDeployment(ctx, deployment.GetId())
	require.NoError(t, err)
	require.True(t, exists, "Deployment should still exist after soft-delete")
	assert.Equal(t, storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED, deleted.GetLifecycleStage())
	assert.NotNil(t, deleted.GetTombstone(), "Deployment should have tombstone")
	assert.NotNil(t, deleted.GetTombstone().GetDeletedAt())
	assert.NotNil(t, deleted.GetTombstone().GetExpiresAt())

	// The mock will verify that Delete was called exactly once.
	// This ensures process indicators are cleaned up during soft-delete.
}

// TestProcessIndicatorNotCleanedOnUpsert verifies that the process filter is NOT
// deleted when a deployment is updated (not deleted).
func TestProcessIndicatorNotCleanedOnUpsert(t *testing.T) {
	testDB := pgtest.ForT(t)
	defer testDB.Close()

	ctx := sac.WithAllAccess(context.Background())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockProcessFilter := processFilterMocks.NewMockFilter(ctrl)

	ds, err := GetTestPostgresDataStoreWithProcessFilter(t, testDB.DB, mockProcessFilter)
	require.NoError(t, err)

	deployment := fixtures.GetDeployment()
	deployment.Id = uuid.NewV4().String()
	deployment.Name = "test-deployment"
	deployment.ClusterId = "test-cluster"
	deployment.LifecycleStage = storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE

	// Expect NO calls to Delete during upsert.
	mockProcessFilter.EXPECT().Delete(gomock.Any()).Times(0)

	// Upsert the deployment.
	require.NoError(t, ds.UpsertDeployment(ctx, deployment))

	// Update the deployment.
	deployment.Name = "updated-name"
	require.NoError(t, ds.UpsertDeployment(ctx, deployment))

	// Verify no calls to Delete occurred.
}
