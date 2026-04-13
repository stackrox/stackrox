//go:build sql_integration

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/deployment/datastore"
	riskManagerMocks "github.com/stackrox/rox/central/risk/manager/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestBackwardCompatibility_ListDeployments verifies that ListDeployments defaults to excluding soft-deleted deployments.
func TestBackwardCompatibility_ListDeployments(t *testing.T) {
	testDB := pgtest.ForT(t)
	defer testDB.Close()

	ctx := sac.WithAllAccess(context.Background())
	ds, err := datastore.GetTestPostgresDataStore(t, testDB.DB)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskManager := riskManagerMocks.NewMockManager(ctrl)

	service := New(ds, nil, nil, nil, nil, riskManager).(*serviceImpl)

	// Create active deployment.
	activeDeployment := &storage.Deployment{
		Id:             uuid.NewV4().String(),
		Name:           "active-deployment",
		ClusterId:      uuid.NewV4().String(),
		Namespace:      "default",
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, activeDeployment))

	// Create soft-deleted deployment.
	now := time.Now()
	deletedDeployment := &storage.Deployment{
		Id:        uuid.NewV4().String(),
		Name:      "deleted-deployment",
		ClusterId: uuid.NewV4().String(),
		Namespace: "default",
		Tombstone: &storage.Tombstone{
			DeletedAt: timestamppb.New(now.Add(-1 * time.Hour)),
			ExpiresAt: timestamppb.New(now.Add(23 * time.Hour)),
		},
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, deletedDeployment))

	// List deployments without any lifecycle filter (default behavior).
	req := &v1.RawQuery{
		Query: "",
	}

	resp, err := service.ListDeployments(ctx, req)
	require.NoError(t, err)

	// Verify only active deployment is returned (backward compatibility).
	require.Len(t, resp.GetDeployments(), 1)
	assert.Equal(t, activeDeployment.GetId(), resp.GetDeployments()[0].GetId())
}

// TestBackwardCompatibility_CountDeployments verifies that CountDeployments defaults to excluding soft-deleted deployments.
func TestBackwardCompatibility_CountDeployments(t *testing.T) {
	testDB := pgtest.ForT(t)
	defer testDB.Close()

	ctx := sac.WithAllAccess(context.Background())
	ds, err := datastore.GetTestPostgresDataStore(t, testDB.DB)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskManager := riskManagerMocks.NewMockManager(ctrl)

	service := New(ds, nil, nil, nil, nil, riskManager).(*serviceImpl)

	// Create active deployment.
	activeDeployment := &storage.Deployment{
		Id:             uuid.NewV4().String(),
		Name:           "active-deployment",
		ClusterId:      uuid.NewV4().String(),
		Namespace:      "default",
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, activeDeployment))

	// Create soft-deleted deployment.
	now := time.Now()
	deletedDeployment := &storage.Deployment{
		Id:        uuid.NewV4().String(),
		Name:      "deleted-deployment",
		ClusterId: uuid.NewV4().String(),
		Namespace: "default",
		Tombstone: &storage.Tombstone{
			DeletedAt: timestamppb.New(now.Add(-1 * time.Hour)),
			ExpiresAt: timestamppb.New(now.Add(23 * time.Hour)),
		},
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, deletedDeployment))

	// Count deployments without any lifecycle filter (default behavior).
	req := &v1.RawQuery{
		Query: "",
	}

	resp, err := service.CountDeployments(ctx, req)
	require.NoError(t, err)

	// Verify only active deployment is counted (backward compatibility).
	assert.Equal(t, int32(1), resp.GetCount())
}

// TestBackwardCompatibility_ExplicitLifecycleFilter verifies that users can override the default filter.
func TestBackwardCompatibility_ExplicitLifecycleFilter(t *testing.T) {
	testDB := pgtest.ForT(t)
	defer testDB.Close()

	ctx := sac.WithAllAccess(context.Background())
	ds, err := datastore.GetTestPostgresDataStore(t, testDB.DB)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskManager := riskManagerMocks.NewMockManager(ctrl)

	service := New(ds, nil, nil, nil, nil, riskManager).(*serviceImpl)

	// Create active deployment.
	activeDeployment := &storage.Deployment{
		Id:             uuid.NewV4().String(),
		Name:           "active-deployment",
		ClusterId:      uuid.NewV4().String(),
		Namespace:      "default",
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, activeDeployment))

	// Create soft-deleted deployment.
	now := time.Now()
	deletedDeployment := &storage.Deployment{
		Id:        uuid.NewV4().String(),
		Name:      "deleted-deployment",
		ClusterId: uuid.NewV4().String(),
		Namespace: "default",
		Tombstone: &storage.Tombstone{
			DeletedAt: timestamppb.New(now.Add(-1 * time.Hour)),
			ExpiresAt: timestamppb.New(now.Add(23 * time.Hour)),
		},
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, deletedDeployment))

	t.Run("explicit DELETED filter returns only deleted deployments", func(t *testing.T) {
		req := &v1.RawQuery{
			Query: "Lifecycle Stage:DEPLOYMENT_DELETED",
		}

		resp, err := service.ListDeployments(ctx, req)
		require.NoError(t, err)

		require.Len(t, resp.GetDeployments(), 1)
		assert.Equal(t, deletedDeployment.GetId(), resp.GetDeployments()[0].GetId())
	})

	t.Run("explicit ACTIVE filter returns only active deployments", func(t *testing.T) {
		req := &v1.RawQuery{
			Query: "Lifecycle Stage:DEPLOYMENT_ACTIVE",
		}

		resp, err := service.ListDeployments(ctx, req)
		require.NoError(t, err)

		require.Len(t, resp.GetDeployments(), 1)
		assert.Equal(t, activeDeployment.GetId(), resp.GetDeployments()[0].GetId())
	})
}
